import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ProgressMessage, Task, Websocket } from '../../services/websocket';
import { SidebarComponent } from '../../components/sidebar/sidebar';
import { StatsComponent } from '../../components/stats/stats';
import { TopbarComponent } from '../../components/topbar/topbar';

@Component({
  selector: 'app-download-manager',
  imports: [CommonModule, FormsModule, SidebarComponent, StatsComponent, TopbarComponent], 
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager implements OnInit {
  urlInput: string = '';
  isModalOpen: boolean = false;

  tasks: Task[] = [];
  selectedTask: Task | null = null;
  
  currentFilter: string = 'All';
  searchText: string = '';

  totalDownloadedBytes: number = 0;
  activeDownloadCount: number = 0;
  completedCount: number = 0;
  errorCount: number = 0;

  constructor(
    private http: HttpClient,
    private wsService: Websocket,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.wsService.historyUpdates$.subscribe((loadedTasks: Task[]) => {
      if (loadedTasks) {
        this.tasks = loadedTasks.map(t => {
          let initialPercent = 0;
          if (t.totalSize > 0) {
            initialPercent = (t.downloaded / t.totalSize) * 100;
            initialPercent = Math.min(initialPercent, 100);
          } else if (t.status === 'Completed') initialPercent = 100;
          return { ...t, progress: initialPercent };
        });
        this.tasks.reverse();
        if (this.tasks.length > 0) this.selectTask(this.tasks[0]);
        this.updateStats();
        this.cdr.detectChanges();
      }
    });

    this.wsService.progressUpdates$.subscribe((msg: ProgressMessage) => {
      this.updateTask(msg);
    });
  }

  openAddModal() { this.isModalOpen = true; }
  closeAddModal() { this.isModalOpen = false; this.urlInput = ''; }

  onSearch(text: string) {
    this.searchText = text.toLowerCase();
  }

  get filteredTasks(): Task[] {
    let result = this.tasks;

    // Filter by Category/Status
    if (this.currentFilter !== 'All') {
      const statusFilters = ['Downloading', 'Paused', 'Completed', 'Queued', 'Error'];
      if (statusFilters.includes(this.currentFilter)) {
        result = result.filter(t => t.status === this.currentFilter);
      } else {
        result = result.filter(t => {
          const n = t.fileName.toLowerCase();
          if (this.currentFilter === 'video') return n.endsWith('.mp4') || n.endsWith('.mkv');
          if (this.currentFilter === 'music') return n.endsWith('.mp3') || n.endsWith('.wav');
          if (this.currentFilter === 'zip') return n.endsWith('.zip') || n.endsWith('.rar');
          if (this.currentFilter === 'exe') return n.endsWith('.exe') || n.endsWith('.msi');
          if (this.currentFilter === 'doc') return n.endsWith('.pdf') || n.endsWith('.doc');
          return false;
        });
      }
    }

    //Filter by search text
    if (this.searchText) {
      result = result.filter(t => 
        t.fileName.toLowerCase().includes(this.searchText) || 
        t.url.toLowerCase().includes(this.searchText)
      );
    }

    return result;
  }

  setFilter(filter: string) {
    this.currentFilter = filter;
    this.selectedTask = null;
  }

  selectTask(task: Task) { this.selectedTask = task; }

  startDownload() {
    if (!this.urlInput) return;
    const url = this.urlInput;
    this.closeAddModal();

    const existing = this.tasks.find(t => t.id === url);
    if (!existing) {
      const newTask: Task = { id: url, url: url, fileName: 'Pending...', status: 'Downloading', progress: 0, totalSize: 0, downloaded: 0 };
      this.tasks.unshift(newTask);
      this.selectTask(newTask);
    } else { existing.status = 'Downloading'; }
    this.http.post('http://localhost:8080/download', { url: url }).subscribe();
    this.updateStats();
  }

  // Individual Actions
  pauseTask(task: Task) {
    event?.stopPropagation(); 
    task.status = 'Paused';
    this.http.post('http://localhost:8080/pause', { url: task.id }).subscribe();
    this.updateStats();
  }

  resumeTask(task: Task) {
    event?.stopPropagation();
    task.status = 'Downloading';
    this.http.post('http://localhost:8080/resume', { url: task.id }).subscribe();
    this.updateStats();
  }

  //Bulk Actions
  resumeAll() {
    this.tasks.forEach(t => {
      if (t.status === 'Paused' || t.status === 'Error') this.resumeTask(t);
    });
  }

  pauseAll() {
    this.tasks.forEach(t => {
      if (t.status === 'Downloading') this.pauseTask(t);
    });
  }

  stopAll() {
    // For now, treat Stop as Pause until backend supports hard stop
    this.pauseAll(); 
    alert("Hard Stop not fully implemented in backend yet. Pausing all.");
  }

  updateTask(msg: ProgressMessage) {
    const task = this.tasks.find(t => t.id === msg.id);
    if (task) {
      task.fileName = msg.fileName;
      task.progress = Math.min(msg.percent, 100);
      if (task.totalSize > 0) task.downloaded = (task.progress / 100) * task.totalSize;
      if (msg.percent >= 100) {
        task.status = 'Completed';
        task.progress = 100;
        this.updateStats();
      }
      this.cdr.detectChanges();
    }
  }

  updateStats() {
    this.activeDownloadCount = this.tasks.filter(t => t.status === 'Downloading').length;
    this.completedCount = this.tasks.filter(t => t.status === 'Completed').length;
    this.totalDownloadedBytes = this.tasks.reduce((acc, t) => acc + (t.downloaded || 0), 0);
  }

  formatBytes(bytes: number, decimals = 2) {
    if (!+bytes) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
  }

  getFileIcon(fileName: string): string {
    if (!fileName) return 'ph-file';
    const lower = fileName.toLowerCase();
    if (lower.endsWith('.iso')) return 'ph-file-archive';
    if (lower.endsWith('.exe') || lower.endsWith('.msi')) return 'ph-package';
    if (lower.endsWith('.zip') || lower.endsWith('.rar')) return 'ph-file-zip';
    if (lower.endsWith('.mp4') || lower.endsWith('.mkv')) return 'ph-film-strip';
    if (lower.endsWith('.mp3') || lower.endsWith('.wav')) return 'ph-music-note';
    if (lower.endsWith('.pdf') || lower.endsWith('.doc')) return 'ph-file-text';
    return 'ph-file';
  }

  getIconColor(fileName: string): string {
    if (!fileName) return '#a1a1aa';
    const lower = fileName.toLowerCase();
    if (lower.endsWith('.iso')) return '#3b82f6'; 
    if (lower.endsWith('.exe')) return '#22c55e'; 
    if (lower.endsWith('.zip')) return '#eab308'; 
    if (lower.endsWith('.mp4')) return '#a855f7'; 
    if (lower.endsWith('.mp3')) return '#ef4444'; 
    return '#a1a1aa';
  }
}