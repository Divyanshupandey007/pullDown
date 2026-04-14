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
  @Input() activeMode: string = 'auto';
  @Output() tabChange = new EventEmitter<string>();
  @Output() openAddModal = new EventEmitter<void>();
  @Output() modeChange = new EventEmitter<string>();

  isCollapsed = false;
  hoveredMode: string | null = null;

  tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'active', label: 'Active Streams', icon: 'analytics' },
    { id: 'completed', label: 'Archive', icon: 'inventory_2' },
    { id: 'settings', label: 'Settings', icon: 'tune' }
  ];

  modes = [
    { id: 'snail', emoji: '🐌', label: 'Snail', desc: 'Bandwidth saver' },
    { id: 'auto', emoji: '⚡', label: 'Auto', desc: 'Balanced' },
    { id: 'turbo', emoji: '🚀', label: 'Turbo', desc: 'Max speed' }
  ];

  toggleSidebar() {
    this.isCollapsed = !this.isCollapsed;
  }

  selectTab(tabId: string) {
    this.tabChange.emit(tabId);
  }

  selectMode(modeId: string) {
    this.modeChange.emit(modeId);
  }
}