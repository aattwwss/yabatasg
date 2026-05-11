function busApp() {
	return {
		searchTerm: '',
		filteredGroups: [],
		groups: [],
		showAddModal: false,
		showConfirmModal: false,
		confirmMessage: '',
		itemToDelete: null,
		deleteType: null,
		newShortcut: {
			service: '',
			stopNumber: '',
			name: '',
			groupName: '',
			newGroupName: '',
			arrivals: []
		},
		toasts: [],
		toastId: 0,
		loading: true,
		pollInterval: null,

		init() {
			this.loadFromLocalStorage();
			this.filteredGroups = [...this.groups];

			this.fetchAllArrivals().then(() => {
				this.loading = false;
			});

			this.$el.addEventListener('update-arrivals', (event) => {
				const { groupIndex, shortcutIndex, arrivals } = event.detail;
				if (this.groups[groupIndex] && this.groups[groupIndex].shortcuts[shortcutIndex]) {
					this.groups[groupIndex].shortcuts[shortcutIndex].arrivals = arrivals;
				}
			});

			this.pollInterval = setInterval(() => {
				this.fetchAllArrivals();
			}, 30000);
		},

		destroy() {
			if (this.pollInterval) {
				clearInterval(this.pollInterval);
			}
		},

		loadFromLocalStorage() {
			const savedData = localStorage.getItem('busAppData');
			if (savedData) {
				try {
					this.groups = JSON.parse(savedData);
				} catch {
					this.groups = [];
				}
			} else {
				this.groups = [];
			}
		},

		saveToLocalStorage() {
			const dataToSave = this.groups.map(group => ({
				...group,
				shortcuts: group.shortcuts.map(shortcut => ({
					...shortcut,
					arrivals: undefined
				}))
			}));
			localStorage.setItem('busAppData', JSON.stringify(dataToSave));
		},

		filterGroups() {
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
					this.groups.push({
						name: groupName,
						shortcuts: []
					});
				}
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
				arrivals: [null, null, null]
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

		deleteShortcut(groupIndex, shortcutIndex) {
			this.itemToDelete = { groupIndex, shortcutIndex };
			this.deleteType = 'shortcut';
			this.confirmMessage = 'Are you sure you want to delete this shortcut?';
			this.showConfirmModal = true;
		},

		deleteGroup(groupIndex) {
			this.itemToDelete = { groupIndex };
			this.deleteType = 'group';
			const groupName = this.groups[groupIndex].name;
			this.confirmMessage = `Are you sure you want to delete the group "${groupName}" and all its shortcuts?`;
			this.showConfirmModal = true;
		},

		confirmDelete() {
			if (this.deleteType === 'shortcut') {
				const { groupIndex, shortcutIndex } = this.itemToDelete;
				this.groups[groupIndex].shortcuts.splice(shortcutIndex, 1);

				if (this.groups[groupIndex].shortcuts.length === 0) {
					this.groups.splice(groupIndex, 1);
				}

				this.showToast('Shortcut deleted successfully', 'success');
			} else if (this.deleteType === 'group') {
				const { groupIndex } = this.itemToDelete;
				this.groups.splice(groupIndex, 1);
				this.showToast('Group deleted successfully', 'success');
			}

			this.saveToLocalStorage();
			this.filteredGroups = [...this.groups];
			this.showConfirmModal = false;
			this.itemToDelete = null;
		},

		exportData() {
			const dataStr = JSON.stringify(this.groups, null, 2);
			const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);

			const linkElement = document.createElement('a');
			linkElement.setAttribute('href', dataUri);
			linkElement.setAttribute('download', 'bus_shortcuts.json');
			linkElement.click();

			this.showToast('Data exported successfully', 'success');
		},

		importData(event) {
			const file = event.target.files[0];
			if (!file) return;

			const reader = new FileReader();
			reader.onload = (e) => {
				try {
					const importedData = JSON.parse(e.target.result);
					if (!Array.isArray(importedData)) {
						this.showToast('Invalid file format. Expected a JSON array.', 'error');
						return;
					}

					for (const group of importedData) {
						if (!group || typeof group.name !== 'string' || !Array.isArray(group.shortcuts)) {
							this.showToast('Invalid file format: each group must have a name and shortcuts array.', 'error');
							return;
						}
						for (const s of group.shortcuts) {
							if (!s || !s.service || !s.stopNumber) {
								this.showToast('Invalid file format: each shortcut must have service and stopNumber.', 'error');
								return;
							}
						}
					}

					this.groups = importedData;
					this.saveToLocalStorage();
					this.filteredGroups = [...this.groups];
					this.loading = true;
					this.fetchAllArrivals().then(() => {
						this.loading = false;
					});
					this.showToast('Data imported successfully!', 'success');
				} catch {
					this.showToast('Error parsing the file. Please check the file format.', 'error');
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
				const response = await fetch(`/api/v1/busArrival?BusStopCode=${shortcut.stopNumber}&ServiceNo=${shortcut.service}`, {
					method: 'GET',
					headers: {
						'Content-Type': 'application/json'
					}
				});

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
					arrivals
				});

			} catch (error) {
				console.error('Error fetching arrival time:', error);
				this.$dispatch('update-arrivals', {
					groupIndex,
					shortcutIndex,
					arrivals: [null, null, null]
				});
				this.showToast(`Error fetching arrival time for Bus ${shortcut.service} at Stop ${shortcut.stopNumber}`, 'error');
			}
		}
	}
}
