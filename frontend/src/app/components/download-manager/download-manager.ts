import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit, ViewChild, ElementRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ProgressMessage, Task, Websocket } from '../../services/websocket';
import { SidebarComponent } from '../../components/sidebar/sidebar';
import { TopbarComponent } from '../../components/topbar/topbar';

@Component({
  selector: 'app-download-manager',
  imports: [CommonModule, FormsModule, SidebarComponent, TopbarComponent], 
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager implements OnInit {
  urlInput: string = '';
  
  tasks: Task[] = [];
  selectedTask: Task | null = null;
  
  activeTab: string = 'dashboard';
  currentFilter: string = 'All';
  searchText: string = '';

  totalDownloadedBytes: number = 0;
  activeDownloadCount: number = 0;
  completedCount: number = 0;
  errorCount: number = 0;

  showModal: boolean = false;
  showDetails: boolean = false;
  detailTitle: string = '--';
  detailProgress: number = 0;
  activeSettingsTab: string = 'general';

  // Settings toggle states
  toggles: Record<string, boolean> = {
    autoStart: true,
    completionAlerts: true,
    portBinding: true,
    encryption: false,
    enableProxy: false,
    forceHttps: true,
    autoExtract: false,
    autoRetry: true,
    scheduler: false,
    notifComplete: true,
    notifError: true,
    soundEffects: false
  };

  toggleSetting(key: string) {
    this.toggles[key] = !this.toggles[key];
  }

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

  @ViewChild('urlInputField') urlInputField!: ElementRef<HTMLInputElement>;
  @ViewChild('folderInput') folderInput!: ElementRef<HTMLInputElement>;

  downloadPath: string = 'C:\\Downloads';

  // Tab switching
  switchTab(tabId: string) {
    this.activeTab = tabId;
    if (tabId === 'settings') this.activeSettingsTab = 'general';
  }

  setSettingsTab(tab: string) {
    this.activeSettingsTab = tab;
  }

  async browseFolder() {
    // Try modern File System Access API first (Chrome/Edge)
    if ('showDirectoryPicker' in window) {
      try {
        const dirHandle = await (window as any).showDirectoryPicker({ mode: 'read' });
        if (dirHandle) {
          this.downloadPath = dirHandle.name;
          this.cdr.detectChanges();
          return;
        }
      } catch (e: any) {
        if (e.name === 'AbortError') return; // User cancelled
      }
    }
    // Fallback: create a temporary file input with webkitdirectory
    const input = document.createElement('input');
    input.type = 'file';
    (input as any).webkitdirectory = true;
    input.onchange = () => {
      if (input.files && input.files.length > 0) {
        const path = (input.files[0] as any).webkitRelativePath;
        if (path) {
          this.downloadPath = path.split('/')[0];
          this.cdr.detectChanges();
        }
      }
    };
    input.click();
  }

  playNotificationSound() {
    try {
      const ctx = new AudioContext();
      // First tone
      const osc1 = ctx.createOscillator();
      const gain1 = ctx.createGain();
      osc1.type = 'sine';
      osc1.frequency.value = 880;
      gain1.gain.setValueAtTime(0.15, ctx.currentTime);
      gain1.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.3);
      osc1.connect(gain1).connect(ctx.destination);
      osc1.start(ctx.currentTime);
      osc1.stop(ctx.currentTime + 0.3);
      // Second tone (higher)
      const osc2 = ctx.createOscillator();
      const gain2 = ctx.createGain();
      osc2.type = 'sine';
      osc2.frequency.value = 1320;
      gain2.gain.setValueAtTime(0.15, ctx.currentTime + 0.15);
      gain2.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.5);
      osc2.connect(gain2).connect(ctx.destination);
      osc2.start(ctx.currentTime + 0.15);
      osc2.stop(ctx.currentTime + 0.5);
    } catch (e) {
      console.warn('Audio not supported', e);
    }
  }

  // Modal
  toggleModal() {
    this.showModal = !this.showModal;
  }

  openAddModal() { 
    this.showModal = true;
    setTimeout(() => {
      if (this.urlInputField) {
        this.urlInputField.nativeElement.focus();
      }
    }, 100);
  }

  closeModal() {
    this.showModal = false;
    this.urlInput = '';
  }

  // Details panel
  showDetailsPanel(task: Task) {
    this.selectedTask = task;
    this.detailTitle = task.fileName;
    this.detailProgress = task.progress || 0;
    this.showDetails = true;
    // Mark active card
    document.querySelectorAll('.download-card').forEach(c => c.classList.remove('active-card'));
  }

  hideDetailsPanel() {
    this.showDetails = false;
  }

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

  get activeTasks(): Task[] {
    return this.tasks.filter(t => t.status === 'Downloading');
  }

  setFilter(filter: string) {
    this.currentFilter = filter;
    this.selectedTask = null;
  }

  selectTask(task: Task) { 
    this.selectedTask = task; 
    this.detailTitle = task.fileName;
    this.detailProgress = task.progress || 0;
  }

  startDownload() {
    if (!this.urlInput) return;
    const url = this.urlInput;
    this.closeModal();
    
    console.log('Starting Download:', url);

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
  pauseTask(task: Task, event?: Event) {
    if (event) event.stopPropagation(); 
    console.log('Pause triggered for:', task.fileName);
    task.status = 'Paused';
    this.http.post('http://localhost:8080/pause', { url: task.id }).subscribe({
      next: () => console.log('Pause request successful'),
      error: (e) => console.error('Pause request failed', e)
    });
    this.updateStats();
  }

  resumeTask(task: Task, event?: Event) {
    if (event) event.stopPropagation();
    console.log('Resume triggered for:', task.fileName);
    task.status = 'Downloading';
    this.http.post('http://localhost:8080/resume', { url: task.id }).subscribe({
      next: () => console.log('Resume request successful'),
      error: (e) => console.error('Resume request failed', e)
    });
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
    this.pauseAll(); 
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
        // Play sound if enabled
        if (this.toggles['soundEffects']) {
          this.playNotificationSound();
        }
      }
      // Update details panel if this task is selected
      if (this.selectedTask && this.selectedTask.id === task.id) {
        this.detailTitle = task.fileName;
        this.detailProgress = task.progress;
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
    if (!fileName) return 'description';
    const lower = fileName.toLowerCase();
    if (lower.endsWith('.iso')) return 'terminal';
    if (lower.endsWith('.exe') || lower.endsWith('.msi')) return 'terminal';
    if (lower.endsWith('.zip') || lower.endsWith('.rar')) return 'folder_zip';
    if (lower.endsWith('.mp4') || lower.endsWith('.mkv')) return 'videocam';
    if (lower.endsWith('.mp3') || lower.endsWith('.wav')) return 'music_note';
    if (lower.endsWith('.pdf') || lower.endsWith('.doc')) return 'description';
    return 'description';
  }

  getFileTag(fileName: string): string {
    if (!fileName) return 'File';
    const lower = fileName.toLowerCase();
    if (lower.endsWith('.iso')) return 'Engine';
    if (lower.endsWith('.exe') || lower.endsWith('.msi')) return 'Program';
    if (lower.endsWith('.zip') || lower.endsWith('.rar')) return 'Archive';
    if (lower.endsWith('.mp4') || lower.endsWith('.mkv')) return 'Media';
    if (lower.endsWith('.mp3') || lower.endsWith('.wav')) return 'Audio';
    if (lower.endsWith('.pdf') || lower.endsWith('.doc')) return 'Document';
    return 'File';
  }
}