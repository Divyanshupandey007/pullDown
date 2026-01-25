import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProgressMessage, Task, Websocket } from '../../services/websocket';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-download-manager',
  imports: [CommonModule, FormsModule],
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager implements OnInit {
  urlInput: string = '';
  // Use the Task interface
  tasks: Task[] = []; 

  constructor(
    private http: HttpClient,
    private wsService: Websocket,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    // 1. Listen for History (Runs once on connect)
    this.wsService.historyUpdates$.subscribe((loadedTasks: Task[]) => {
      if (loadedTasks) {
        this.tasks = loadedTasks.map(t => {
          // Calculate initial progress % from saved bytes
          let initialPercent = 0;
          if (t.totalSize > 0) {
            initialPercent = (t.downloaded / t.totalSize) * 100;
            initialPercent = Math.min(initialPercent,100);
          } else if (t.status === 'Completed') {
            initialPercent = 100;
          }
          
          return { ...t, progress: initialPercent };
        });
        
        // Reverse so newest is at top (optional)
        this.tasks.reverse(); 
        this.cdr.detectChanges();
      }
    });

    // 2. Listen for Live Updates
    this.wsService.progressUpdates$.subscribe((msg: ProgressMessage) => {
      this.updateTask(msg);
    });
  }

  startDownload() {
    if (!this.urlInput) return;
    const url = this.urlInput;

    // Check if already exists to avoid duplicates in UI
    const existing = this.tasks.find(t => t.id === url);
    if (!existing) {
      // Add placeholder
      this.tasks.unshift({
        id: url,
        url: url,
        fileName: 'Pending...',
        status: 'Downloading',
        progress: 0,
        totalSize: 0,
        downloaded: 0
      });
    } else {
      existing.status = 'Downloading';
    }

    this.http.post('http://localhost:8080/download', { url: url }).subscribe();
    this.urlInput = ''; 
  }

  pauseTask(task: Task) {
    task.status = 'Paused';
    this.http.post('http://localhost:8080/pause', { url: task.id }).subscribe();
  }

  resumeTask(task: Task) {
    task.status = 'Downloading';
    this.http.post('http://localhost:8080/resume', { url: task.id }).subscribe();
  }

  updateTask(msg: ProgressMessage) {
    const task = this.tasks.find(t => t.id === msg.id);
    if (task) {
      task.fileName = msg.fileName;
      task.progress = Math.min(msg.percent,100);
      
      if (msg.percent >= 100) {
        task.status = 'Completed';
        task.progress=100
      }
      this.cdr.detectChanges();
    }
  }
}
