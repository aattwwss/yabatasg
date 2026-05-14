function busApp() {
    const STORAGE_KEY = 'busAppData';
    const THEME_KEY = 'busAppTheme';
    const POLL_MS = 30000;
    const STALE_MS = 60000;

    return {
        groups: [],
        filteredGroups: [],
        searchTerm: '',
        loading: true,
        toasts: [],
        theme: 'light',

        // modals
        showAddModal: false,
        showConfirmModal: false,
        confirmMsg: '',
        confirmAction: null,

        // form
        form: { stopNumber: '', name: '', groupName: '', newGroupName: '' },

        // swipe
        swiped: null,
        swipedDir: null,
        _touchTarget: null,
        _touchStartX: 0,
        _touchCurX: 0,
        _swipeThreshold: 60,

        // edit
        editTarget: null,

        _pollTimer: null,
        _toastId: 0,
        _failCount: {},

        // nearby
        nearbyView: '',
        nearbyStops: [],
        selectedStop: null,
        geoError: '',
        nearbyLoading: false,
        nearbySearch: '',
        filteredNearbyStops: [],

        // sync
        AUTH_TOKEN_KEY: 'busAppToken',
        showSyncModal: false,
        syncView: '',
        authToken: '',
        syncPhrase: '',
        linkWords: ['', '', '', ''],
        linkError: '',
        _syncDebounce: null,

        _modalScrollLock(open) {
            if (open && !document.body.classList.contains('modal-open')) {
                const y = window.scrollY;
                document.body.dataset.scrollY = y;
                document.body.style.top = `-${y}px`;
                document.body.classList.add('modal-open');
            } else if (!open && document.body.classList.contains('modal-open')) {
                document.body.classList.remove('modal-open');
                const y = Math.abs(parseInt(document.body.style.top) || 0);
                document.body.style.top = '';
                window.scrollTo(0, y);
            }
        },

        init() {
            this._loadTheme();
            this._loadAuth();
            this._load();
            this.filteredGroups = [...this.groups];
            this._fetchAll().then(() => { this.loading = false; this._backfillStops(); });
            if (this.authToken) {
                this._loadFromServer();
            }
            this.$el.addEventListener('update-services', e => {
                const { groupIndex, shortcutIndex, services, fetchedAt } = e.detail;
                const s = this.groups[groupIndex]?.shortcuts[shortcutIndex];
                if (s) { s.services = services; s.lastFetched = fetchedAt; }
            });
            this._pollTimer = setInterval(() => this._fetchAll(), POLL_MS);
            this._onPopStateBound = this._onPopState.bind(this);
            window.addEventListener('popstate', this._onPopStateBound);
            this.$watch('showAddModal', val => { if (!val) this.editTarget = null; });
        },

        destroy() {
            clearInterval(this._pollTimer);
            window.removeEventListener('popstate', this._onPopStateBound);
        },

        _onPopState(e) {
            if (e.state?.appView) {
                this.nearbyView = e.state.appView;
                if (e.state.appView === 'stopDetail' && e.state.code) {
                    this.selectedStop = { code: e.state.code, roadName: e.state.roadName, services: [], loading: true, error: '' };
                    this._loadStopDetail(e.state.code, e.state.roadName);
                }
                if (e.state.appView === 'shortcutDetail' && e.state.gi !== undefined) {
                    const s = this.groups[e.state.gi]?.shortcuts[e.state.si];
                    if (s) {
                        this.selectedStop = { code: s.stopNumber, roadName: s.name || s.roadName || `Stop ${s.stopNumber}`, services: s.services, loading: false, error: '' };
                    }
                }
            } else {
                this.nearbyView = '';
                this.selectedStop = null;
            }
        },

        // ── Theme ──
        _loadTheme() {
            const saved = localStorage.getItem(THEME_KEY);
            this.theme = saved || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
            document.documentElement.setAttribute('data-theme', this.theme);
        },

        toggleTheme() {
            this.theme = this.theme === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', this.theme);
            localStorage.setItem(THEME_KEY, this.theme);
        },

        // ── Persistence ──
        _load() {
            try {
                const raw = localStorage.getItem(STORAGE_KEY);
                if (raw) this.groups = JSON.parse(raw);
            } catch { this.groups = []; }
            for (const g of this.groups) {
                for (const s of g.shortcuts) {
                    // Migrate old per-service shortcuts to per-stop format
                    if (s.service !== undefined) {
                        s.services = [{
                            serviceNo: s.service,
                            operator: '',
                            next1: (s.arrivals && s.arrivals[0] != null) ? s.arrivals[0] : null,
                            next2: (s.arrivals && s.arrivals[1] != null) ? s.arrivals[1] : null,
                            next3: (s.arrivals && s.arrivals[2] != null) ? s.arrivals[2] : null
                        }];
                        delete s.service;
                        delete s.arrivals;
                    }
                    if (!s.services) s.services = [];
                    s.roadName ??= '';
                    s.description ??= '';
                    s.lastFetched ??= 0;
                }
            }
        },

        _save() {
            const data = this.groups.map(g => ({
                name: g.name,
                shortcuts: g.shortcuts.map(s => ({ stopNumber: s.stopNumber, name: s.name, roadName: s.roadName, description: s.description }))
            }));
            localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
            if (this.authToken) this._syncToServer();
        },

        // ── Search ──
        filter() {
            this._closeSwipe();
            const t = this.searchTerm.toLowerCase().trim();
            if (!t) { this.filteredGroups = [...this.groups]; return; }
            this.filteredGroups = this.groups.reduce((acc, g) => {
                const matches = g.shortcuts.filter(s =>
                    s.name.toLowerCase().includes(t) ||
                    s.stopNumber.includes(t) ||
                    s.roadName?.toLowerCase().includes(t) ||
                    s.description?.toLowerCase().includes(t) ||
                    (s.services || []).some(svc => svc.serviceNo.includes(t))
                );
                if (g.name.toLowerCase().includes(t) || matches.length) {
                    acc.push({ ...g, shortcuts: matches.length ? matches : g.shortcuts });
                }
                return acc;
            }, []);
        },

        // ── Add ──
        async _backfillStops() {
            for (const g of this.groups) {
                for (const s of g.shortcuts) {
                    if (!s.roadName) {
                        const info = await this._lookupStop(s.stopNumber);
                        if (info) { s.roadName = info.roadName; s.description = info.description; }
                    }
                }
            }
            this._save();
        },

        async _lookupStop(code) {
            try {
                const r = await fetch(`/api/v1/stops/${code}`);
                if (r.ok) return await r.json();
            } catch {}
            return null;
        },

        async add() {
            this._closeSwipe();
            const f = this.form;
            if (!f.stopNumber) { this._toast('Fill in the stop number', 'error'); return; }

            // Edit path
            if (this.editTarget) {
                const { gi, si } = this.editTarget;
                const oldGroup = this.groups[gi];
                if (!oldGroup || !oldGroup.shortcuts[si]) { this.editTarget = null; return; }

                let targetGroupName = f.groupName;
                if (targetGroupName === '__new') {
                    if (!f.newGroupName.trim()) { this._toast('Enter a group name', 'error'); return; }
                    targetGroupName = f.newGroupName.trim();
                    if (!this.groups.some(g => g.name === targetGroupName)) {
                        this.groups.push({ name: targetGroupName, shortcuts: [] });
                    }
                }
                if (!targetGroupName) { this._toast('Select a group', 'error'); return; }

                const targetGroup = this.groups.find(g => g.name === targetGroupName);
                const isSameGroup = targetGroup === oldGroup;
                const hasDup = isSameGroup
                    ? oldGroup.shortcuts.some((x, i) => i !== si && x.stopNumber === f.stopNumber)
                    : targetGroup.shortcuts.some(x => x.stopNumber === f.stopNumber);
                if (hasDup) { this._toast('Already exists in this group', 'error'); return; }

                const s = oldGroup.shortcuts[si];
                const oldCode = s.stopNumber;
                s.stopNumber = f.stopNumber;
                s.name = f.name.trim();
                s.services = [];
                s.lastFetched = 0;
                if (s.stopNumber !== oldCode || !s.roadName) {
                    const info = await this._lookupStop(f.stopNumber);
                    if (info) { s.roadName = info.roadName; s.description = info.description; }
                }

                if (!isSameGroup) {
                    oldGroup.shortcuts.splice(si, 1);
                    targetGroup.shortcuts.push(s);
                    if (oldGroup.shortcuts.length === 0) {
                        this.groups.splice(this.groups.indexOf(oldGroup), 1);
                    }
                }

                this._save();
                this.showAddModal = false;
                this._resetForm();
                this.editTarget = null;
                this.filteredGroups = [...this.groups];
                this._toast('Shortcut updated', 'success');
                this._fetchAll();
                return;
            }

            let groupName = f.groupName;
            if (groupName === '__new') {
                if (!f.newGroupName.trim()) { this._toast('Enter a group name', 'error'); return; }
                groupName = f.newGroupName.trim();
                if (!this.groups.some(g => g.name === groupName)) {
                    this.groups.push({ name: groupName, shortcuts: [] });
                }
            }
            if (!groupName) { this._toast('Select a group', 'error'); return; }

            const group = this.groups.find(g => g.name === groupName);
            if (!group) { this._toast('Group not found', 'error'); return; }

            if (group.shortcuts.some(s => s.stopNumber === f.stopNumber)) {
                this._toast('Already exists in this group', 'error'); return;
            }

            const shortcut = {
                stopNumber: f.stopNumber,
                name: f.name.trim(),
                services: [],
                roadName: '',
                description: '',
                lastFetched: 0
            };
            const info = await this._lookupStop(f.stopNumber);
            if (info) { shortcut.roadName = info.roadName; shortcut.description = info.description; }
            group.shortcuts.push(shortcut);
            const gi = this.groups.indexOf(group);
            const si = group.shortcuts.length - 1;

            this._save();
            this.showAddModal = false;
            this._resetForm();
            this.filteredGroups = [...this.groups];
            this._toast('Shortcut added', 'success');
            this._fetchOne(f.stopNumber, [{ gi, si }]);
        },

        _resetForm() {
            this.form = { stopNumber: '', name: '', groupName: '', newGroupName: '' };
        },

        // ── Delete ──
        deleteShortcut(fgi, fsi) {
            this._closeSwipe();
            const s = this.filteredGroups[fgi]?.shortcuts[fsi];
            if (!s) return;
            for (const g of this.groups) {
                const idx = g.shortcuts.indexOf(s);
                if (idx !== -1) {
                    g.shortcuts.splice(idx, 1);
                    if (!g.shortcuts.length) this.groups.splice(this.groups.indexOf(g), 1);
                    break;
                }
            }
            this._save();
            this.filteredGroups = [...this.groups];
            this._toast('Shortcut deleted', 'success');
        },

        askDeleteGroup(gi) {
            this.confirmMsg = `Delete group "${this.groups[gi].name}" and all its shortcuts?`;
            this.confirmAction = () => {
                this.groups.splice(gi, 1);
                this._save();
                this.filteredGroups = [...this.groups];
                this.showConfirmModal = false;
                this.confirmAction = null;
                this._toast('Group deleted', 'success');
            };
            this.showConfirmModal = true;
        },

        // ── Edit ──
        edit(fgi, fsi) {
            this._closeSwipe();
            const s = this.filteredGroups[fgi]?.shortcuts[fsi];
            if (!s) return;
            let foundGi = -1, foundSi = -1;
            for (const [i, g] of this.groups.entries()) {
                const idx = g.shortcuts.indexOf(s);
                if (idx !== -1) { foundGi = i; foundSi = idx; break; }
            }
            if (foundGi === -1) return;
            this.editTarget = { gi: foundGi, si: foundSi };
            const group = this.groups[foundGi];
            this.form.stopNumber = s.stopNumber;
            this.form.name = s.name || '';
            this.form.groupName = group.name;
            this.form.newGroupName = '';
            this.showAddModal = true;
        },

        // ── Swipe ──
        touchStart(e, gi, si) {
            if (this.swiped && this.swiped.gi === gi && this.swiped.si === si) { this._closeSwipe(); return; }
            this._closeSwipe();
            this._touchTarget = { gi, si };
            this._touchStartX = e.touches[0].clientX;
            this._touchCurX = this._touchStartX;
        },
        touchMove(e) { this._touchCurX = e.touches[0].clientX; },
        touchEnd() {
            const diff = this._touchStartX - this._touchCurX;
            const t = this._touchTarget;
            this._touchTarget = null;
            if (!t) return;
            if (diff > this._swipeThreshold) { this.swiped = t; this.swipedDir = 'left'; }
            else if (diff < -this._swipeThreshold) { this.swiped = t; this.swipedDir = 'right'; }
        },
        _closeSwipe() {
            this.swiped = null;
            this.swipedDir = null;
            this._touchTarget = null;
            this._touchStartX = this._touchCurX = 0;
        },

        // ── Drag-and-drop (via Alpine Sort plugin) ──
        _onSort(filteredGi, item, position) {
            const fgroup = this.filteredGroups[filteredGi];
            if (!fgroup) return;
            const group = this.groups.find(g => g.name === fgroup.name);
            if (!group) return;
            const stopNumber = item;
            const oldIndex = group.shortcuts.findIndex(s => s.stopNumber === stopNumber);
            if (oldIndex === -1) return;
            // $position counts the group-header DOM child at index 0, so it's
            // off by one relative to the 0-indexed shortcuts array.
            const idx = position - 1;
            if (oldIndex === idx) return;
            const [moved] = group.shortcuts.splice(oldIndex, 1);
            group.shortcuts.splice(idx, 0, moved);
            this._save();
        },

        // ── Arrivals ──
        arrivalClass(v) {
            if (v == null || v < 0) return '';
            if (v <= 2) return 'urgent';
            if (v <= 8) return 'soon';
            return 'later';
        },
        formatArrival(v) {
            if (v == null || v < 0) return '--';
            return v + '';
        },
        isStale(s) { return s.lastFetched && (Date.now() - s.lastFetched) > STALE_MS; },

        relativeTime(ts) {
            const sec = Math.floor((Date.now() - ts) / 1000);
            if (sec < 5) return 'just now';
            if (sec < 60) return `${sec}s ago`;
            const min = Math.floor(sec / 60);
            if (min < 60) return `${min}m ago`;
            return `${Math.floor(min / 60)}h ago`;
        },

        // ── Nearby ──
        showNearby() {
            this.nearbyView = 'stops';
            this.geoError = '';
            this.nearbyStops = [];
            this.filteredNearbyStops = [];
            this.nearbyLoading = true;
            this.nearbySearch = '';
            history.pushState({ appView: 'stops' }, '');

            if (!navigator.geolocation) {
                this.geoError = 'Geolocation not supported by your browser';
                this.nearbyLoading = false;
                return;
            }

            navigator.geolocation.getCurrentPosition(
                pos => this._loadNearby(pos.coords.latitude, pos.coords.longitude),
                err => { console.error('Geolocation error:', err); this.geoError = err.message; this.nearbyLoading = false; },
                { timeout: 10000, maximumAge: 60000 }
            );
        },

        async _loadNearby(lat, lng) {
            try {
                const r = await fetch(`/api/v1/stops/nearby?lat=${lat}&lng=${lng}&limit=20`);
                if (!r.ok) throw new Error(`HTTP ${r.status}`);
                this.nearbyStops = await r.json();
                this.filteredNearbyStops = [...this.nearbyStops];
            } catch {
                this.geoError = 'Failed to load nearby stops';
            }
            this.nearbyLoading = false;
        },

        applyNearbyFilter() {
            const q = this.nearbySearch.toLowerCase().trim();
            if (!q) {
                this.filteredNearbyStops = [...this.nearbyStops];
                return;
            }
            this.filteredNearbyStops = this.nearbyStops.filter(s =>
                s.code.includes(q) ||
                s.roadName.toLowerCase().includes(q) ||
                s.description.toLowerCase().includes(q)
            );
        },

        async selectStop(code, roadName) {
            this.nearbyView = 'stopDetail';
            this.selectedStop = { code, roadName, services: [], loading: true, error: '' };
            history.pushState({ appView: 'stopDetail', code, roadName }, '');
            this._loadStopDetail(code, roadName);
        },

        async _loadStopDetail(code, roadName) {
            try {
                const r = await fetch(`/api/v1/stops/${code}/arrivals`);
                if (!r.ok) throw new Error(`HTTP ${r.status}`);
                const data = await r.json();
                this.selectedStop.services = data.services || [];
                this.selectedStop.loading = false;
            } catch {
                this.selectedStop.error = 'Failed to load arrivals';
                this.selectedStop.loading = false;
            }
        },

        async addShortcutFromStop(stopCode) {
            this.form.stopNumber = stopCode;
            this.form.name = '';
            this.form.groupName = this.groups.length > 0 ? this.groups[0].name : '';
            this.nearbyView = '';
            this.showAddModal = true;
            const info = await this._lookupStop(stopCode);
            if (info) { this.form.name = info.roadName || ''; }
        },

        viewShortcutDetail(gi, si) {
            const s = this.groups[gi]?.shortcuts[si];
            if (!s) return;
            this._closeSwipe();
            this.nearbyView = 'shortcutDetail';
            this.selectedStop = {
                code: s.stopNumber,
                roadName: s.name || s.roadName || `Stop ${s.stopNumber}`,
                services: s.services,
                loading: false,
                error: ''
            };
            history.pushState({ appView: 'shortcutDetail', gi, si }, '');
        },

        backToNearby() { history.back(); },
        backToHome() { history.back(); },

        async _fetchAll() {
            const stopMap = new Map();
            for (const [gi, g] of this.groups.entries()) {
                for (const [si, s] of g.shortcuts.entries()) {
                    if (!stopMap.has(s.stopNumber)) stopMap.set(s.stopNumber, []);
                    stopMap.get(s.stopNumber).push({ gi, si });
                }
            }
            const jobs = [];
            for (const [stopNumber, targets] of stopMap) {
                jobs.push(this._fetchOne(stopNumber, targets));
            }
            await Promise.allSettled(jobs);
        },

        async _fetchOne(stopNumber, targets) {
            const key = stopNumber;
            try {
                const r = await fetch(`/api/v1/stops/${stopNumber}/arrivals`);
                if (!r.ok) throw new Error(`HTTP ${r.status}`);
                const data = await r.json();
                const services = data.services || [];
                for (const { gi, si } of targets) {
                    this.$dispatch('update-services', { groupIndex: gi, shortcutIndex: si, services, fetchedAt: Date.now() });
                }
                this._failCount[key] = 0;
            } catch {
                const services = [];
                for (const { gi, si } of targets) {
                    this.$dispatch('update-services', { groupIndex: gi, shortcutIndex: si, services, fetchedAt: Date.now() });
                }
                this._failCount[key] = (this._failCount[key] || 0) + 1;
                if (this._failCount[key] === 3) {
                    this._toast(`Cannot reach server for stop ${stopNumber}`, 'error');
                }
            }
        },

        // ── Import / Export ──
        exportData() {
            const a = document.createElement('a');
            a.href = 'data:application/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(this.groups, null, 2));
            a.download = 'bus_shortcuts.json';
            a.click();
            this._toast('Exported', 'success');
        },

        importData(e) {
            const file = e.target.files[0];
            if (!file) return;
            const reader = new FileReader();
            reader.onload = ev => {
                try {
                    const data = JSON.parse(ev.target.result);
                    if (!Array.isArray(data)) { this._toast('Invalid: expected array', 'error'); return; }
                    for (const g of data) {
                        if (!g || typeof g.name !== 'string' || !Array.isArray(g.shortcuts)) {
                            this._toast('Invalid group format', 'error'); return;
                        }
                        for (const s of g.shortcuts) {
                            if (!s?.stopNumber) { this._toast('Invalid shortcut format', 'error'); return; }
                            if (s.service !== undefined) {
                                s.services = [{ serviceNo: s.service, operator: '', next1: null, next2: null, next3: null }];
                                delete s.service;
                                delete s.arrivals;
                            }
                            if (!s.services) s.services = [];
                            s.roadName ??= '';
                            s.description ??= '';
                            s.lastFetched = 0;
                        }
                    }
                    this.groups = data;
                    this._save();
                    this.filteredGroups = [...this.groups];
                    this.loading = true;
                    this._fetchAll().then(() => { this.loading = false; });
                    this._toast('Imported', 'success');
                } catch { this._toast('Could not parse file', 'error'); }
            };
            reader.readAsText(file);
        },

        // ── Toast ──
        _toast(msg, type) {
            const id = ++this._toastId;
            this.toasts.push({ id, msg, type });
            setTimeout(() => { this.toasts = this.toasts.filter(t => t.id !== id); }, 3000);
        },

        // ── Sync ──
        _loadAuth() {
            this.authToken = localStorage.getItem(this.AUTH_TOKEN_KEY) || '';
        },

        _saveAuth() {
            if (this.authToken) {
                localStorage.setItem(this.AUTH_TOKEN_KEY, this.authToken);
            } else {
                localStorage.removeItem(this.AUTH_TOKEN_KEY);
            }
        },

        openSync() {
            this.linkWords = ['', '', '', ''];
            this.linkError = '';
            if (this.authToken) {
                this.syncView = 'synced';
                this.syncPhrase = localStorage.getItem('busAppPhrase') || '';
            } else {
                this.syncView = '';
            }
            this.showSyncModal = true;
        },

        async createAccount() {
            this.syncView = 'syncing';
            this._serializeGroups();
            const data = JSON.stringify(this.groups.map(g => ({
                name: g.name,
                shortcuts: g.shortcuts.map(s => ({ stopNumber: s.stopNumber, name: s.name, roadName: s.roadName, description: s.description }))
            })));
            try {
                const r = await fetch('/api/v1/auth/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ config: data === '[]' ? '' : data })
                });
                if (!r.ok) throw new Error((await r.json()).error || 'Failed');
                const j = await r.json();
                this.authToken = j.token;
                this.syncPhrase = j.phrase;
                this.syncView = 'created';
                this._saveAuth();
                localStorage.setItem('busAppPhrase', j.phrase);
            } catch (e) {
                this.syncView = '';
                this._toast('Failed to create account', 'error');
            }
        },

        async linkDevice() {
            const phrase = this.linkWords.map(w => w.trim().toLowerCase().replace(/[^a-z]/g, '')).join('-');
            if (phrase.split('-').filter(Boolean).length < 4) { this.linkError = 'Enter all 4 words'; return; }
            this.syncView = 'syncing';
            this.linkError = '';
            try {
                const r = await fetch('/api/v1/auth/link', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ phrase })
                });
                if (!r.ok) {
                    const msg = (await r.json()).error || 'Not found';
                    this.linkError = msg;
                    this.syncView = 'link';
                    return;
                }
                const j = await r.json();
                this.authToken = j.token;
                this._saveAuth();
                // Fetch server config.
                const cr = await fetch('/api/v1/config', {
                    headers: { 'Authorization': 'Bearer ' + j.token }
                });
                if (cr.ok) {
                    const cfg = await cr.json();
                    if (Array.isArray(cfg) && cfg.length > 0) {
                        this.groups = cfg;
                        for (const g of this.groups) {
                            for (const s of g.shortcuts) {
                                if (s.service !== undefined) {
                                    s.services = [{ serviceNo: s.service, operator: '', next1: null, next2: null, next3: null }];
                                    delete s.service;
                                }
                                if (!s.services) s.services = [];
                                s.roadName ??= '';
                                s.description ??= '';
                                s.lastFetched = 0;
                            }
                        }
                        this._serializeGroups();
                        localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
                        this.filteredGroups = [...this.groups];
                        this.loading = true;
                        this._fetchAll().then(() => { this.loading = false; });
                    }
                }
                // Fetch phrase for display.
                const mr = await fetch('/api/v1/auth/me', {
                    headers: { 'Authorization': 'Bearer ' + j.token }
                });
                if (mr.ok) {
                    const me = await mr.json();
                    this.syncPhrase = me.phrase;
                    localStorage.setItem('busAppPhrase', me.phrase);
                }
                this.syncView = 'synced';
                this._toast('Device linked', 'success');
            } catch {
                this.linkError = 'Connection failed';
                this.syncView = 'link';
            }
        },

        async unlinkDevice() {
            try {
                await fetch('/api/v1/config', {
                    method: 'DELETE',
                    headers: { 'Authorization': 'Bearer ' + this.authToken }
                });
            } catch { /* best effort */ }
            this.authToken = '';
            this.syncPhrase = '';
            this.syncView = '';
            this.showSyncModal = false;
            this._saveAuth();
            localStorage.removeItem('busAppPhrase');
            this._toast('Device unlinked', 'info');
        },

        _phraseInput(idx, evt) {
            let val = evt.target.value;
            if (!val) return;

            // Handle full phrase typed/pasted with dashes.
            const parts = val.split('-');
            if (parts.length >= 2 && parts.filter(p => p.length >= 2).length >= 2) {
                for (let i = 0; i < 4; i++) {
                    this.linkWords[i] = (parts[i] || '').replace(/[^a-zA-Z]/g, '').toLowerCase();
                }
                this._focusNext(evt.target, Math.min(parts.filter(Boolean).length, 3));
                return;
            }

            // Strip non-alpha and lowercase.
            const cleaned = val.replace(/[^a-zA-Z]/g, '').toLowerCase();
            if (cleaned !== val) {
                evt.target.value = cleaned;
                this.linkWords[idx] = cleaned;
            }
        },

        _phraseKeydown(idx, evt) {
            if (evt.key === 'Enter') {
                evt.preventDefault();
                this.linkDevice();
                return;
            }
            if (evt.key === ' ' || evt.key === '-') {
                evt.preventDefault();
                if (idx < 3) this._focusNext(evt.target, idx + 1);
                return;
            }
            if (evt.key === 'Backspace' && evt.target.value === '' && idx > 0) {
                this._focusNext(evt.target, idx - 1);
            }
        },

        _phrasePaste(evt) {
            const paste = (evt.clipboardData || window.clipboardData).getData('text');
            if (!paste || !paste.includes('-')) return;
            evt.preventDefault();
            const parts = paste.trim().toLowerCase().split('-');
            for (let i = 0; i < 4; i++) {
                this.linkWords[i] = (parts[i] || '').replace(/[^a-z]/g, '');
            }
            this._focusNext(evt.target, Math.min(parts.length, 3));
        },

        _focusNext(fromEl, idx) {
            const row = fromEl.closest('.phrase-inputs');
            if (!row) return;
            const inputs = row.querySelectorAll('input');
            if (inputs[idx]) {
                inputs[idx].focus();
                inputs[idx].select();
            }
        },

        copyPhrase() {
            navigator.clipboard.writeText(this.syncPhrase).then(
                () => this._toast('Copied', 'success'),
                () => this._toast('Failed to copy', 'error')
            );
        },

        _serializeGroups() {
            // Ensure groups/shortcuts are plain objects (not Alpine proxies).
            this.groups = JSON.parse(JSON.stringify(
                this.groups.map(g => ({
                    name: g.name,
                    shortcuts: g.shortcuts.map(s => ({
                        stopNumber: s.stopNumber,
                        name: s.name,
                        roadName: s.roadName || '',
                        description: s.description || '',
                        services: s.services || [],
                        lastFetched: s.lastFetched || 0
                    }))
                }))
            ));
        },


        async _syncToServer() {
            const data = this.groups.map(g => ({
                name: g.name,
                shortcuts: g.shortcuts.map(s => ({ stopNumber: s.stopNumber, name: s.name, roadName: s.roadName, description: s.description }))
            }));
            try {
                await fetch('/api/v1/config', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.authToken
                    },
                    body: JSON.stringify(data)
                });
            } catch { /* silent */ }
        },

        async _loadFromServer() {
            try {
                const r = await fetch('/api/v1/auth/me', {
                    headers: { 'Authorization': 'Bearer ' + this.authToken }
                });
                if (!r.ok) { this.authToken = ''; this._saveAuth(); return; }
                const me = await r.json();
                this.syncPhrase = me.phrase;
                localStorage.setItem('busAppPhrase', me.phrase);

                const cr = await fetch('/api/v1/config', {
                    headers: { 'Authorization': 'Bearer ' + this.authToken }
                });
                if (cr.ok) {
                    const cfg = await cr.json();
                    if (Array.isArray(cfg) && cfg.length > 0) {
                        this.groups = cfg;
                        for (const g of this.groups) {
                            for (const s of g.shortcuts) {
                                if (s.service !== undefined) {
                                    s.services = [{ serviceNo: s.service, operator: '', next1: null, next2: null, next3: null }];
                                    delete s.service;
                                }
                                if (!s.services) s.services = [];
                                s.roadName ??= '';
                                s.description ??= '';
                                s.lastFetched = 0;
                            }
                        }
                        localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
                        this.filteredGroups = [...this.groups];
                        this.loading = true;
                        this._fetchAll().then(() => { this.loading = false; });
                    }
                }
            } catch { /* offline — use localStorage */ }
        },

        // ── Helpers ──
        onlyDigits(e) { e.target.value = e.target.value.replace(/[^0-9]/g, ''); },
    };
}
