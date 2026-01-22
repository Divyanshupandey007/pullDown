import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProgressMessage, Websocket } from '../../services/websocket';
import { CommonModule } from '@angular/common';

interface DownloadTask {
  id: string;       // URL
  fileName: string;
  progress: number;
  status: 'Downloading' | 'Paused' | 'Completed' | 'Error';
}

@Component({
  selector: 'app-download-manager',
  imports: [CommonModule, FormsModule],
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager implements OnInit {
  urlInput: string = '';
  tasks: DownloadTask[] = []; // List of all downloads

  constructor(
    private http: HttpClient,
    private wsService: Websocket,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    // Subscribe to WS updates
    this.wsService.progressUpdates$.subscribe((msg: ProgressMessage) => {
      this.updateTask(msg);
    });
  }

  startDownload() {
    if (!this.urlInput) return;

    const url = this.urlInput;
    
    // Add to list immediately for UI feedback
    if (!this.tasks.find(t => t.id === url)) {
      this.tasks.unshift({
        id: url,
        fileName: 'Pending...',
        progress: 0,
        status: 'Downloading'
      });
    } else {
        // If resuming a completed/paused one, just update status
        const existing = this.tasks.find(t => t.id === url);
        if(existing) existing.status = 'Downloading';
    }

    this.http.post('http://localhost:8080/download', { url: url }).subscribe();
    this.urlInput = ''; // Clear input
  }

  pauseTask(task: DownloadTask) {
    task.status = 'Paused';
    this.http.post('http://localhost:8080/pause', { url: task.id }).subscribe();
  }

  resumeTask(task: DownloadTask) {
    task.status = 'Downloading';
    this.http.post('http://localhost:8080/resume', { url: task.id }).subscribe();
  }

  updateTask(msg: ProgressMessage) {
    // Find the row that matches the ID
    const task = this.tasks.find(t => t.id === msg.id);
    if (task) {
      task.fileName = msg.fileName;
      task.progress = msg.percent;
      
      if (msg.percent >= 100) {
        task.status = 'Completed';
      }
      // Trigger UI Refresh
      this.cdr.detectChanges();
    }
  }
}
