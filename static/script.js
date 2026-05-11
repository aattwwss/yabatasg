function busApp() {
	return {
		searchTerm: '',
		filteredGroups: [],
		groups: [],
		showAddModal: false,
		showConfirmModal: false,
		confirmMessage: '',
		itemToDelete: null,
		newShortcut: {
			service: '',
			stopNumber: '',
			name: '',
			groupName: '',
			newGroupName: ''
		},
		toasts: [],
		toastId: 0,
		loading: true,
		pollInterval: null,

		// Swipe state
		swipedCard: null,
		touchStartX: 0,
		touchCurrentX: 0,
		swipeThreshold: 60,

		init() {
			this.loadFromLocalStorage();
			this.filteredGroups = [...this.groups];

			this.fetchAllArrivals().then(() => {
				this.loading = false;
			});

			this.$el.addEventListener('update-arrivals', (event) => {
				const { groupIndex, shortcutIndex, arrivals, fetchedAt } = event.detail;
				if (this.groups[groupIndex] && this.groups[groupIndex].shortcuts[shortcutIndex]) {
					this.groups[groupIndex].shortcuts[shortcutIndex].arrivals = arrivals;
					this.groups[groupIndex].shortcuts[shortcutIndex].lastFetched = fetchedAt;
				}
			});

			this.pollInterval = setInterval(() => {
				this.fetchAllArrivals();
			}, 30000);
		},

		destroy() {
			if (this.pollInterval) clearInterval(this.pollInterval);
		},

		loadFromLocalStorage() {
			const savedData = localStorage.getItem('busAppData');
			if (savedData) {
				try {
					this.groups = JSON.parse(savedData);
				} catch {
					this.groups = [];
				}
			}
			for (const group of this.groups) {
				for (const s of group.shortcuts) {
					if (!s.arrivals) s.arrivals = [null, null, null];
					if (!s.lastFetched) s.lastFetched = 0;
				}
			}
		},

		saveToLocalStorage() {
			const dataToSave = this.groups.map(group => ({
				name: group.name,
				shortcuts: group.shortcuts.map(s => ({
					service: s.service,
					stopNumber: s.stopNumber,
					name: s.name
				}))
			}));
			localStorage.setItem('busAppData', JSON.stringify(dataToSave));
		},

		filterGroups() {
			this.closeSwipe();
			if (!this.searchTerm) {
				this.filteredGroups = [...this.groups];
				return;
			}
			const term = this.searchTerm.toLowerCase();
			this.filteredGroups = this.groups.reduce((acc, group) => {
				const matchingShortcuts = group.shortcuts.filter(s =>
					s.name.toLowerCase().includes(term) ||
					s.service.includes(term) ||
					s.stopNumber.includes(term)
				);
				if (group.name.toLowerCase().includes(term) || matchingShortcuts.length > 0) {
					acc.push({
						...group,
						shortcuts: matchingShortcuts.length > 0 ? matchingShortcuts : group.shortcuts
					});
				}
				return acc;
			}, []);
		},

		addShortcut() {
			this.closeSwipe();
			if (!this.newShortcut.service || !this.newShortcut.stopNumber) {
				this.showToast('Please enter both bus service and stop number', 'error');
				return;
			}

			let groupName = this.newShortcut.groupName;
			if (groupName === 'new') {
				if (!this.newShortcut.newGroupName) {
					this.showToast('Please enter a name for the new group', 'error');
					return;
				}
				groupName = this.newShortcut.newGroupName;
				if (!this.groups.some(g => g.name === groupName)) {
					this.groups.push({ name: groupName, shortcuts: [] });
				}
			}

			if (!groupName) {
				this.showToast('Please select a group', 'error');
				return;
			}

			const group = this.groups.find(g => g.name === groupName);
			if (group) {
				const duplicate = group.shortcuts.find(s =>
					s.service === this.newShortcut.service &&
					s.stopNumber === this.newShortcut.stopNumber
				);
				if (duplicate) {
					this.showToast('This bus service and stop already exists in this group', 'error');
					return;
				}
			}

			const shortcut = {
				service: this.newShortcut.service,
				stopNumber: this.newShortcut.stopNumber,
				name: this.newShortcut.name || `Bus ${this.newShortcut.service} - Stop ${this.newShortcut.stopNumber}`,
				arrivals: [null, null, null],
				lastFetched: 0
			};

			if (group) {
				group.shortcuts.push(shortcut);
			}

			const groupIndex = this.groups.findIndex(g => g.name === groupName);
			const shortcutIndex = group.shortcuts.length - 1;

			this.saveToLocalStorage();
			this.showAddModal = false;
			this.resetForm();
			this.filteredGroups = [...this.groups];
			this.showToast('Shortcut added', 'success');

			this.fetchArrivalTime(shortcut, groupIndex, shortcutIndex);
		},

		resetForm() {
			this.newShortcut = {
				service: '',
				stopNumber: '',
				name: '',
				groupName: '',
				newGroupName: ''
			};
		},

		// Swipe handlers
		onTouchStart(event, groupIndex, shortcutIndex) {
			if (this.swipedCard &&
				this.swipedCard.group === groupIndex &&
				this.swipedCard.shortcut === shortcutIndex) {
				this.closeSwipe();
				return;
			}
			this.closeSwipe();
			this._touchTarget = { group: groupIndex, shortcut: shortcutIndex };
			this.touchStartX = event.touches[0].clientX;
			this.touchCurrentX = this.touchStartX;
		},

		onTouchMove(event) {
			this.touchCurrentX = event.touches[0].clientX;
		},

		onTouchEnd() {
			const diff = this.touchStartX - this.touchCurrentX;
			const target = this._touchTarget;
			this._touchTarget = null;

			if (!target) return;

			if (diff > this.swipeThreshold) {
				this.swipedCard = target;
			} else if (diff < -this.swipeThreshold) {
				this.closeSwipe();
			} else {
				// Small or no movement — leave current state as-is
			}
		},

		closeSwipe() {
			this.swipedCard = null;
			this._touchTarget = null;
			this.touchStartX = 0;
			this.touchCurrentX = 0;
		},

		deleteShortcutNow(filteredGroupIndex, filteredShortcutIndex) {
			this.closeSwipe();

			const filtered = this.filteredGroups[filteredGroupIndex];
			if (!filtered) return;
			const shortcut = filtered.shortcuts[filteredShortcutIndex];
			if (!shortcut) return;

			for (const group of this.groups) {
				const idx = group.shortcuts.indexOf(shortcut);
				if (idx !== -1) {
					group.shortcuts.splice(idx, 1);
					if (group.shortcuts.length === 0) {
						this.groups.splice(this.groups.indexOf(group), 1);
					}
					break;
				}
			}

			this.saveToLocalStorage();
			this.filteredGroups = [...this.groups];
			this.showToast('Shortcut deleted', 'success');
		},

		// Group deletion with confirmation
		deleteGroup(groupIndex) {
			this.itemToDelete = groupIndex;
			const groupName = this.groups[groupIndex].name;
			this.confirmMessage = `Delete the group "${groupName}" and all its shortcuts?`;
			this.showConfirmModal = true;
		},

		confirmDelete() {
			this.closeSwipe();
			const groupIndex = this.itemToDelete;
			this.groups.splice(groupIndex, 1);
			this.saveToLocalStorage();
			this.filteredGroups = [...this.groups];
			this.showConfirmModal = false;
			this.itemToDelete = null;
			this.showToast('Group deleted', 'success');
		},

		// Arrival styling
		isStale(shortcut) {
			return shortcut.lastFetched && (Date.now() - shortcut.lastFetched) > 60000;
		},

		arrivalClass(minutes, position) {
			if (minutes == null) return '';
			if (minutes <= 2) return 'urgent';
			if (minutes <= 8) return 'soon';
			return 'later';
		},

		exportData() {
			const dataStr = JSON.stringify(this.groups, null, 2);
			const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);
			const a = document.createElement('a');
			a.setAttribute('href', dataUri);
			a.setAttribute('download', 'bus_shortcuts.json');
			a.click();
			this.showToast('Data exported', 'success');
		},

		importData(event) {
			const file = event.target.files[0];
			if (!file) return;
			const reader = new FileReader();
			reader.onload = (e) => {
				try {
					const importedData = JSON.parse(e.target.result);
					if (!Array.isArray(importedData)) {
						this.showToast('Invalid format. Expected a JSON array.', 'error');
						return;
					}
					for (const group of importedData) {
						if (!group || typeof group.name !== 'string' || !Array.isArray(group.shortcuts)) {
							this.showToast('Invalid format: each group must have name and shortcuts.', 'error');
							return;
						}
						for (const s of group.shortcuts) {
							if (!s || !s.service || !s.stopNumber) {
								this.showToast('Invalid format: each shortcut must have service and stopNumber.', 'error');
								return;
							}
							s.arrivals = [null, null, null];
							s.lastFetched = 0;
						}
					}
					this.groups = importedData;
					this.saveToLocalStorage();
					this.filteredGroups = [...this.groups];
					this.loading = true;
					this.fetchAllArrivals().then(() => { this.loading = false; });
					this.showToast('Data imported', 'success');
				} catch {
					this.showToast('Error parsing file.', 'error');
				}
			};
			reader.readAsText(file);
		},

		showToast(message, type = 'info') {
			const id = ++this.toastId;
			this.toasts.push({ id, message, type });
			setTimeout(() => {
				this.toasts = this.toasts.filter(toast => toast.id !== id);
			}, 3000);
		},

		async fetchAllArrivals() {
			const promises = [];
			for (const [groupIndex, group] of this.groups.entries()) {
				for (const [shortcutIndex, shortcut] of group.shortcuts.entries()) {
					promises.push(this.fetchArrivalTime(shortcut, groupIndex, shortcutIndex));
				}
			}
			await Promise.allSettled(promises);
		},

		async fetchArrivalTime(shortcut, groupIndex, shortcutIndex) {
			try {
				const response = await fetch(
					`/api/v1/busArrival?BusStopCode=${shortcut.stopNumber}&ServiceNo=${shortcut.service}`,
					{ method: 'GET', headers: { 'Content-Type': 'application/json' } }
				);

				if (!response.ok) {
					throw new Error(`API request failed with status ${response.status}`);
				}

				const data = await response.json();

				let arrivals;
				if (Array.isArray(data) && data.length >= 3) {
					arrivals = data;
				} else {
					arrivals = [null, null, null];
				}

				this.$dispatch('update-arrivals', {
					groupIndex,
					shortcutIndex,
					arrivals,
					fetchedAt: Date.now()
				});

			} catch (error) {
				console.error('Error fetching arrival time:', error);
				this.$dispatch('update-arrivals', {
					groupIndex,
					shortcutIndex,
					arrivals: [null, null, null],
					fetchedAt: Date.now()
				});
				this.showToast(`Error fetching Bus ${shortcut.service} at Stop ${shortcut.stopNumber}`, 'error');
			}
		}
	}
}
