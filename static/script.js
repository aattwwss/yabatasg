function busApp() {
    const STORAGE_KEY = 'busAppData';
    const POLL_MS = 30000;
    const STALE_MS = 60000;

    return {
        groups: [],
        filteredGroups: [],
        searchTerm: '',
        loading: true,
        toasts: [],

        // modals
        showAddModal: false,
        showConfirmModal: false,
        confirmMsg: '',
        confirmAction: null,

        // form
        form: { service: '', stopNumber: '', name: '', groupName: '', newGroupName: '' },

        // swipe
        swiped: null,
        _touchTarget: null,
        _touchStartX: 0,
        _touchCurX: 0,
        _swipeThreshold: 60,
        _pollTimer: null,
        _toastId: 0,

        init() {
            this._load();
            this.filteredGroups = [...this.groups];
            this._fetchAll().then(() => { this.loading = false; });
            this.$el.addEventListener('update-arrivals', e => {
                const { groupIndex, shortcutIndex, arrivals, fetchedAt } = e.detail;
                const s = this.groups[groupIndex]?.shortcuts[shortcutIndex];
                if (s) { s.arrivals = arrivals; s.lastFetched = fetchedAt; }
            });
            this._pollTimer = setInterval(() => this._fetchAll(), POLL_MS);
        },

        destroy() { clearInterval(this._pollTimer); },

        // ── Persistence ──
        _load() {
            try {
                const raw = localStorage.getItem(STORAGE_KEY);
                if (raw) this.groups = JSON.parse(raw);
            } catch { this.groups = []; }
            for (const g of this.groups) {
                for (const s of g.shortcuts) {
                    if (!s.arrivals) s.arrivals = [null, null, null];
                    s.lastFetched ??= 0;
                }
            }
        },

        _save() {
            const data = this.groups.map(g => ({
                name: g.name,
                shortcuts: g.shortcuts.map(s => ({ service: s.service, stopNumber: s.stopNumber, name: s.name }))
            }));
            localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
        },

        // ── Search ──
        filter() {
            this._closeSwipe();
            const t = this.searchTerm.toLowerCase().trim();
            if (!t) { this.filteredGroups = [...this.groups]; return; }
            this.filteredGroups = this.groups.reduce((acc, g) => {
                const matches = g.shortcuts.filter(s =>
                    s.name.toLowerCase().includes(t) ||
                    s.service.includes(t) ||
                    s.stopNumber.includes(t)
                );
                if (g.name.toLowerCase().includes(t) || matches.length) {
                    acc.push({ ...g, shortcuts: matches.length ? matches : g.shortcuts });
                }
                return acc;
            }, []);
        },

        // ── Add ──
        add() {
            this._closeSwipe();
            const f = this.form;
            if (!f.service || !f.stopNumber) { this._toast('Fill in both service and stop number', 'error'); return; }

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

            if (group.shortcuts.some(s => s.service === f.service && s.stopNumber === f.stopNumber)) {
                this._toast('Already exists in this group', 'error'); return;
            }

            const shortcut = {
                service: f.service,
                stopNumber: f.stopNumber,
                name: f.name.trim() || `Bus ${f.service} - Stop ${f.stopNumber}`,
                arrivals: [null, null, null],
                lastFetched: 0
            };
            group.shortcuts.push(shortcut);
            const gi = this.groups.indexOf(group);
            const si = group.shortcuts.length - 1;

            this._save();
            this.showAddModal = false;
            this._resetForm();
            this.filteredGroups = [...this.groups];
            this._toast('Shortcut added', 'success');
            this._fetchOne(shortcut, gi, si);
        },

        _resetForm() {
            this.form = { service: '', stopNumber: '', name: '', groupName: '', newGroupName: '' };
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
            if (diff > this._swipeThreshold) this.swiped = t;
            else if (diff < -this._swipeThreshold) this._closeSwipe();
        },
        _closeSwipe() {
            this.swiped = null;
            this._touchTarget = null;
            this._touchStartX = this._touchCurX = 0;
        },

        // ── Arrivals ──
        arrivalClass(v) {
            if (v == null) return '';
            if (v <= 2) return 'urgent';
            if (v <= 8) return 'soon';
            return 'later';
        },
        isStale(s) { return s.lastFetched && (Date.now() - s.lastFetched) > STALE_MS; },

        async _fetchAll() {
            const jobs = [];
            for (const [gi, g] of this.groups.entries()) {
                for (const [si, s] of g.shortcuts.entries()) jobs.push(this._fetchOne(s, gi, si));
            }
            await Promise.allSettled(jobs);
        },

        async _fetchOne(s, gi, si) {
            try {
                const r = await fetch(`/api/v1/busArrival?BusStopCode=${s.stopNumber}&ServiceNo=${s.service}`);
                if (!r.ok) throw new Error(`HTTP ${r.status}`);
                const data = await r.json();
                const arrivals = Array.isArray(data) && data.length >= 3 ? data : [null, null, null];
                this.$dispatch('update-arrivals', { groupIndex: gi, shortcutIndex: si, arrivals, fetchedAt: Date.now() });
            } catch {
                this.$dispatch('update-arrivals', { groupIndex: gi, shortcutIndex: si, arrivals: [null, null, null], fetchedAt: Date.now() });
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
                            if (!s?.service || !s?.stopNumber) { this._toast('Invalid shortcut format', 'error'); return; }
                            s.arrivals = [null, null, null];
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

        // ── Helpers ──
        onlyDigits(e) { e.target.value = e.target.value.replace(/[^0-9]/g, ''); },
    };
}
