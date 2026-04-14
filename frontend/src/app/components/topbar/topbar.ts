import { Component, Output, EventEmitter, OnInit, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-topbar',
  imports: [CommonModule, FormsModule],
  templateUrl: './topbar.html',
  styleUrls: ['./topbar.css']
})
export class TopbarComponent implements OnInit {
  @Input() aggregateSpeed: number = 0;
  @Input() taskCount: number = 0;
  @Input() currentToast: { message: string; icon: string; color: string; removing?: boolean } | null = null;
  @Output() search = new EventEmitter<string>();
  @Output() addUrl = new EventEmitter<void>();
  @Output() resumeAll = new EventEmitter<void>();
  @Output() pauseAll = new EventEmitter<void>();
  @Output() stopAll = new EventEmitter<void>();

  searchText: string = '';
  showNotifications = false;
  showAccountMenu = false;
  showAboutModal = false;
  showAppearanceModal = false;
  activeTheme: string = 'dark';

  @Input() notifications: { title: string; detail: string; time: Date }[] = [];

  ngOnInit() {
    const saved = localStorage.getItem('pulldown-theme') || 'dark';
    this.activeTheme = saved;
    this.applyTheme(saved);
  }

  onSearchChange() {
    this.search.emit(this.searchText);
  }

  toggleNotifications(event: Event) {
    event.stopPropagation();
    this.showNotifications = !this.showNotifications;
    this.showAccountMenu = false;
  }

  clearNotifications() {
    this.notifications = [];
  }

  toggleAccountMenu(event: Event) {
    event.stopPropagation();
    this.showAccountMenu = !this.showAccountMenu;
    this.showNotifications = false;
  }

  openAppearance() {
    this.showAccountMenu = false;
    this.showAppearanceModal = true;
  }

  closeAppearance() {
    this.showAppearanceModal = false;
  }

  setAppTheme(theme: string) {
    this.activeTheme = theme;
    this.applyTheme(theme);
    localStorage.setItem('pulldown-theme', theme);
  }

  private applyTheme(theme: string) {
    document.documentElement.setAttribute('data-theme', theme);
  }

  openAbout() {
    this.showAccountMenu = false;
    this.showAboutModal = true;
  }

  closeAbout() {
    this.showAboutModal = false;
  }

  formatSpeed(bytesPerSec: number): { value: string; unit: string } {
    if (!bytesPerSec || bytesPerSec <= 0) return { value: '0', unit: 'B/s' };
    const k = 1024;
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
    return {
      value: parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(1)).toString(),
      unit: sizes[i]
    };
  }
}