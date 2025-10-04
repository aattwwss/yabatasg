function busApp() {
	return {
		searchTerm: '',
		filteredGroups: [],
		groups: [],
		showAddModal: false,
		showConfirmModal: false,
		confirmMessage: '',
		itemToDelete: null,
		deleteType: null, // 'shortcut' or 'group'
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

		init() {
			this.loadFromLocalStorage();
			this.filteredGroups = [...this.groups];

			// Fetch all arrivals after loading data
			this.fetchAllArrivals();

			// Listen for arrival updates
			this.$el.addEventListener('update-arrivals', (event) => {
				const { groupIndex, shortcutIndex, arrivals } = event.detail;
				this.groups[groupIndex].shortcuts[shortcutIndex].arrivals = arrivals;
				this.saveToLocalStorage();
			});
		},

		loadFromLocalStorage() {
			const savedData = localStorage.getItem('busAppData');
			if (savedData) {
				this.groups = JSON.parse(savedData);
			} else {
				// Initialize with sample data
				this.groups = [];
			}
		},

		saveToLocalStorage() {
			// Create a copy without arrival times
			const dataToSave = this.groups.map(group => ({
				...group,
				shortcuts: group.shortcuts.map(shortcut => ({
					...shortcut,
					arrivals: undefined // Exclude arrivals from storage
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
			this.filteredGroups = this.groups.filter(group =>
				group.name.toLowerCase().includes(term)
			);
		},

		addShortcut() {
			// Validate inputs
			if (!this.newShortcut.service || !this.newShortcut.stopNumber) {
				this.showToast('Please enter both bus service and stop number', 'error');
				return;
			}

			// Determine group name
			let groupName = this.newShortcut.groupName;
			if (groupName === 'new') {
				if (!this.newShortcut.newGroupName) {
					this.showToast('Please enter a name for the new group', 'error');
					return;
				}
				groupName = this.newShortcut.newGroupName;

				// Create new group if it doesn't exist
				if (!this.groups.some(g => g.name === groupName)) {
					this.groups.push({
						name: groupName,
						shortcuts: []
					});
				}
			}

			// Check if the same service and stop already exists in the group
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

			// // Generate mock arrival times
			// const arrivals = [
			// 	Math.floor(Math.random() * 10) + 1,
			// 	Math.floor(Math.random() * 15) + 10,
			// 	Math.floor(Math.random() * 20) + 20
			// ];

			// Create new shortcut
			const shortcut = {
				service: this.newShortcut.service,
				stopNumber: this.newShortcut.stopNumber,
				name: this.newShortcut.name || `Bus ${this.newShortcut.service} - Stop ${this.newShortcut.stopNumber}`,
				arrivals: [null, null, null]
			};

			// Add to the appropriate group
			if (group) {
				group.shortcuts.push(shortcut);
			}

			// Save and reset
			this.saveToLocalStorage();
			this.showAddModal = false;
			this.resetForm();
			this.filteredGroups = [...this.groups];

			// Fetch arrival time for the new shortcut
			this.fetchArrivalTime(shortcut, this.filteredGroups.length - 1, group.shortcuts.length - 1).then(() => {
				this.showToast('Shortcut added successfully', 'success');
			});
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

				// If the group is now empty, remove it
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

			const exportFileDefaultName = 'bus_shortcuts.json';

			const linkElement = document.createElement('a');
			linkElement.setAttribute('href', dataUri);
			linkElement.setAttribute('download', exportFileDefaultName);
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
					if (Array.isArray(importedData)) {
						this.groups = importedData;
						this.saveToLocalStorage();
						this.filteredGroups = [...this.groups];
						this.showToast('Data imported successfully!', 'success');
					} else {
						this.showToast('Invalid file format. Please import a valid JSON file.', 'error');
					}
				} catch (error) {
					this.showToast('Error parsing the file. Please check the file format.', 'error');
				}
			};
			reader.readAsText(file);
		},

		showToast(message, type = 'info') {
			const id = ++this.toastId;
			this.toasts.push({ id, message, type });

			// Remove toast after 3 seconds
			setTimeout(() => {
				this.toasts = this.toasts.filter(toast => toast.id !== id);
			}, 3000);
		},

		async fetchAllArrivals() {
			for (const [groupIndex, group] of this.groups.entries()) {
				for (const [shortcutIndex, shortcut] of group.shortcuts.entries()) {
					await this.fetchArrivalTime(shortcut, groupIndex, shortcutIndex);
				}
			}
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

				// Use Alpine's reactivity system
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
					arrivals: [0, 0, 0]
				});
				this.showToast(`Error fetching arrival time for Bus ${shortcut.service} at Stop ${shortcut.stopNumber}`, 'error');
			}
		}
	}
}
