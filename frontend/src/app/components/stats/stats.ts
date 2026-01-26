import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Task } from '../../services/websocket';

@Component({
  selector: 'app-stats',
  imports: [CommonModule],
  templateUrl: './stats.html',
  styleUrls: ['./stats.css']
})
export class StatsComponent {
  @Input() selectedTask: Task | null = null;
  @Input() totalBytes: number = 0;
  @Input() activeCount: number = 0;
  @Input() completedCount: number = 0;
  @Input() errorCount: number = 0;

  // Helper for formatting
  formatBytes(bytes: number, decimals = 2) {
    if (!+bytes) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
  }
}