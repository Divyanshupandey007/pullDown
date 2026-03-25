import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-sidebar',
  imports: [CommonModule],
  templateUrl: './sidebar.html',
  styleUrls: ['./sidebar.css']
})
export class SidebarComponent {
  @Input() activeTab: string = 'dashboard';
  @Output() tabChange = new EventEmitter<string>();
  @Output() openAddModal = new EventEmitter<void>();

  isCollapsed = false;

  tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'active', label: 'Active Streams', icon: 'analytics' },
    { id: 'completed', label: 'Archive', icon: 'inventory_2' },
    { id: 'settings', label: 'Settings', icon: 'tune' }
  ];

  toggleSidebar() {
    this.isCollapsed = !this.isCollapsed;
  }

  selectTab(tabId: string) {
    this.tabChange.emit(tabId);
  }
}