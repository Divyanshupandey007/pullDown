import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Task } from '../../services/websocket';

@Component({
  selector: 'app-sidebar',
  imports: [CommonModule],
  templateUrl: './sidebar.html',
  styleUrls: ['./sidebar.css']
})
export class SidebarComponent {
  @Input() tasks: Task[] = [];
  @Input() activeFilter: string = 'All';
  @Input() storageUsed: number = 0;
  
  @Output() filterChange = new EventEmitter<string>();

  categories = [
    { id: 'All', label: 'All Downloads', icon: 'ph-folder-open' },
    { id: 'doc', label: 'Documents', icon: 'ph-file-text' },
    { id: 'video', label: 'Videos', icon: 'ph-film-strip' },
    { id: 'music', label: 'Music', icon: 'ph-music-note' },
    { id: 'zip', label: 'Compressed', icon: 'ph-file-zip' },
    { id: 'exe', label: 'Programs', icon: 'ph-package' }
  ];

  statuses = [
    { id: 'Downloading', icon: 'ph-download-simple', color: '#3b82f6' },
    { id: 'Paused', icon: 'ph-pause', color: '#eab308' },
    { id: 'Completed', icon: 'ph-check-circle', color: '#22c55e' },
    { id: 'Queued', icon: 'ph-clock', color: '#a1a1aa' },
    { id: 'Error', icon: 'ph-warning-circle', color: '#ef4444' }
  ];

  selectFilter(filter: string) {
    this.filterChange.emit(filter);
  }

  // --- Counters ---
  countStatus(status: string): number {
    return this.tasks.filter(t => t.status === status).length;
  }

  countType(type: string): number {
    if (type === 'All') return this.tasks.length;
    return this.tasks.filter(t => this.checkType(t.fileName, type)).length;
  }

  // Helper logic for file types
  private checkType(name: string, type: string): boolean {
    const n = name.toLowerCase();
    if (type === 'video') return n.endsWith('.mp4') || n.endsWith('.mkv') || n.endsWith('.avi');
    if (type === 'music') return n.endsWith('.mp3') || n.endsWith('.wav') || n.endsWith('.flac');
    if (type === 'zip') return n.endsWith('.zip') || n.endsWith('.rar') || n.endsWith('.7z');
    if (type === 'exe') return n.endsWith('.exe') || n.endsWith('.msi') || n.endsWith('.iso');
    if (type === 'doc') return n.endsWith('.pdf') || n.endsWith('.doc') || n.endsWith('.txt');
    return false;
  }

  // Helper for Storage Display
  formatBytes(bytes: number) {
    if (!+bytes) return '0 B';
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${['B', 'KB', 'MB', 'GB', 'TB'][i]}`;
  }
}