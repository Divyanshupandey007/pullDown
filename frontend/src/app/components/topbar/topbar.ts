import { Component, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-topbar',
  imports: [CommonModule, FormsModule],
  templateUrl: './topbar.html',
  styleUrls: ['./topbar.css']
})
export class TopbarComponent implements OnInit {
  @Output() search = new EventEmitter<string>();
  @Output() addUrl = new EventEmitter<void>();
  @Output() resumeAll = new EventEmitter<void>();
  @Output() pauseAll = new EventEmitter<void>();
  @Output() stopAll = new EventEmitter<void>();
  @Output() themeChange = new EventEmitter<string>();

  searchText: string = '';
  activeMode: string = 'auto';
  showNotifications = false;
  showAccountMenu = false;
  showAboutModal = false;
  showAppearanceModal = false;
  activeTheme: string = 'dark';

  notifications: { title: string; detail: string }[] = [];

  ngOnInit() {
    const saved = localStorage.getItem('pulldown-theme') || 'dark';
    this.activeTheme = saved;
    this.applyTheme(saved);
  }

  onSearchChange() {
    this.search.emit(this.searchText);
  }

  setTheme(mode: string) {
    this.activeMode = mode;
    const themes: Record<string, { accent: string; glow: string }> = {
      'snail': { accent: '#facc15', glow: 'rgba(250, 204, 21, 0.15)' },
      'auto': { accent: '#fb923c', glow: 'rgba(251, 146, 60, 0.15)' },
      'turbo': { accent: '#4ade80', glow: 'rgba(74, 222, 128, 0.15)' }
    };
    const config = themes[mode];
    document.documentElement.style.setProperty('--accent-color', config.accent);
    document.documentElement.style.setProperty('--glow-color', config.glow);
    this.themeChange.emit(mode);
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
}