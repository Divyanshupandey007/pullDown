import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit, ViewChild, ElementRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ErrorMessage, ProgressMessage, Task, Websocket } from '../../services/websocket';
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
  aggregateSpeed: number = 0;
  notifications: { title: string; detail: string; time: Date }[] = [];
  speedHistory: number[] = new Array(60).fill(0);
  peakSpeed: number = 0;

  @ViewChild('bandwidthCanvas') bandwidthCanvas!: ElementRef<HTMLCanvasElement>;

  // Toast system
  currentToast: { message: string; icon: string; color: string; removing?: boolean } | null = null;
  private toastTimeout: any;
  private chartDrawScheduled = false;
  isDragging: boolean = false;
  activeMode: string = 'auto';

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
    this.saveSettings();
  }

  saveSettings() {
    const payload = {
      downloadPath: this.downloadPath,
      maxDownloads: this.maxDownloads,
      maxConnections: this.maxConnections,
      connTimeout: this.connTimeout,
      proxyHost: this.proxyHost,
      proxyPort: this.proxyPort,
      autoStart: this.toggles['autoStart'],
      completionAlert: this.toggles['completionAlerts'],
      portBinding: this.toggles['portBinding'],
      encryption: this.toggles['encryption'],
      enableProxy: this.toggles['enableProxy'],
      forceHttps: this.toggles['forceHttps'],
      autoExtract: this.toggles['autoExtract'],
      autoRetry: this.toggles['autoRetry'],
      scheduler: this.toggles['scheduler'],
      notifComplete: this.toggles['notifComplete'],
      notifError: this.toggles['notifError'],
      soundEffects: this.toggles['soundEffects']
    };
    this.http.post('http://localhost:8080/settings', payload).subscribe({
      next: () => console.log('Settings saved'),
      error: (err) => console.error('Failed to save settings:', err)
    });
  }

  constructor(
    private http: HttpClient,
    private wsService: Websocket,
    private cdr: ChangeDetectorRef
  ) { }

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

    // Handle download errors from backend
    this.wsService.errorUpdates$.subscribe((msg: ErrorMessage) => {
      const task = this.tasks.find(t => t.id === msg.id);
      if (task) {
        task.status = 'Error';
        task.speed = 0;
        this.updateStats();
        this.cdr.detectChanges();
        this.showToast(`${task.fileName} — ${msg.message}`, 'error', '#ef4444');
      }
    });

    // Load settings from backend
    this.http.get<any>('http://localhost:8080/settings').subscribe({
      next: (s) => {
        this.downloadPath = s.downloadPath;
        this.maxDownloads = s.maxDownloads;
        this.maxConnections = s.maxConnections;
        this.connTimeout = s.connTimeout;
        this.proxyHost = s.proxyHost || '';
        this.proxyPort = s.proxyPort || 8080;
        this.toggles = {
          autoStart: s.autoStart,
          completionAlerts: s.completionAlert,
          portBinding: s.portBinding,
          encryption: s.encryption,
          enableProxy: s.enableProxy,
          forceHttps: s.forceHttps,
          autoExtract: s.autoExtract,
          autoRetry: s.autoRetry,
          scheduler: s.scheduler,
          notifComplete: s.notifComplete,
          notifError: s.notifError,
          soundEffects: s.soundEffects
        };
        this.cdr.detectChanges();
      },
      error: (err) => console.error('Failed to load settings:', err)
    });
  }

  @ViewChild('urlInputField') urlInputField!: ElementRef<HTMLInputElement>;
  @ViewChild('folderInput') folderInput!: ElementRef<HTMLInputElement>;

  downloadPath: string = 'C:\\Downloads';
  maxDownloads: number = 4;
  maxConnections: number = 16;
  connTimeout: number = 30;
  proxyHost: string = '';
  proxyPort: number = 8080;

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
          this.saveSettings();
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
          this.saveSettings();
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
    this.showToast('Download started', 'download', 'var(--accent-color)');
    this.updateStats();
  }

  // Individual Actions
  pauseTask(task: Task, event?: Event) {
    if (event) event.stopPropagation();
    task.status = 'Paused';
    task.speed = 0;
    task.eta = 0;
    this.http.post('http://localhost:8080/pause', { url: task.id }).subscribe();
    this.showToast(task.fileName + ' paused', 'pause', '#eab308');
    this.updateStats();
  }

  resumeTask(task: Task, event?: Event) {
    if (event) event.stopPropagation();
    task.status = 'Downloading';
    this.http.post('http://localhost:8080/resume', { url: task.id }).subscribe();
    this.showToast(task.fileName + ' resumed', 'play_arrow', '#4ade80');
    this.updateStats();
  }

  deleteTask(task: Task, event?: Event) {
    if (event) event.stopPropagation();
    const name = task.fileName;
    this.http.delete('http://localhost:8080/delete', { body: { url: task.id } }).subscribe({
      next: () => {
        this.tasks = this.tasks.filter(t => t.id !== task.id);
        if (this.selectedTask?.id === task.id) {
          this.selectedTask = null;
          this.showDetails = false;
        }
        this.updateStats();
        this.showToast(name + ' deleted', 'delete', '#f87171');
        this.cdr.detectChanges();
      },
      error: (e) => console.error('Delete failed', e)
    });
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
      if (msg.totalSize > 0) task.totalSize = msg.totalSize;
      if (task.totalSize > 0) task.downloaded = (task.progress / 100) * task.totalSize;
      task.speed = msg.speed || 0;
      // Use backend-provided ETA (more accurate than client-side calculation)
      task.eta = msg.eta || 0;
      if (msg.percent >= 100) {
        task.status = 'Completed';
        task.progress = 100;
        task.speed = 0;
        task.eta = 0;
        this.updateStats();
        // Push notification if enabled
        if (this.toggles['notifComplete']) {
          this.notifications.unshift({
            title: '✅ Download Complete',
            detail: task.fileName + ' — ' + this.formatBytes(task.totalSize),
            time: new Date()
          });
        }
        // Play sound if enabled
        if (this.toggles['soundEffects']) {
          this.playNotificationSound();
        }
      }
      // Update aggregate speed
      this.aggregateSpeed = this.tasks
        .filter(t => t.status === 'Downloading')
        .reduce((sum, t) => sum + (t.speed || 0), 0);
      // Track speed history for bandwidth chart
      this.speedHistory.push(this.aggregateSpeed);
      if (this.speedHistory.length > 60) this.speedHistory.shift();
      if (this.aggregateSpeed > this.peakSpeed) this.peakSpeed = this.aggregateSpeed;
      // Throttle chart redraws — schedule one per animation frame
      if (!this.chartDrawScheduled) {
        this.chartDrawScheduled = true;
        requestAnimationFrame(() => {
          this.drawBandwidthChart();
          this.chartDrawScheduled = false;
        });
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

  drawBandwidthChart() {
    if (!this.bandwidthCanvas?.nativeElement) return;
    const canvas = this.bandwidthCanvas.nativeElement;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Handle HiDPI
    const rect = canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    const w = rect.width;
    const h = rect.height;
    const padding = { top: 10, bottom: 5, left: 0, right: 0 };
    const chartW = w - padding.left - padding.right;
    const chartH = h - padding.top - padding.bottom;

    ctx.clearRect(0, 0, w, h);

    const data = this.speedHistory;
    const maxVal = Math.max(...data, 1);

    // Draw grid lines (subtle)
    ctx.strokeStyle = 'rgba(128, 128, 128, 0.08)';
    ctx.lineWidth = 1;
    for (let i = 1; i <= 3; i++) {
      const y = padding.top + (chartH / 4) * i;
      ctx.beginPath();
      ctx.moveTo(padding.left, y);
      ctx.lineTo(w - padding.right, y);
      ctx.stroke();
    }

    // Build points
    const points: { x: number; y: number }[] = [];
    for (let i = 0; i < data.length; i++) {
      const x = padding.left + (i / (data.length - 1)) * chartW;
      const y = padding.top + chartH - (data[i] / maxVal) * chartH;
      points.push({ x, y });
    }

    if (points.length < 2) return;

    // Draw smooth curve using quadratic bezier
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length; i++) {
      const prev = points[i - 1];
      const curr = points[i];
      const cpx = (prev.x + curr.x) / 2;
      ctx.quadraticCurveTo(prev.x, prev.y, cpx, (prev.y + curr.y) / 2);
    }
    const last = points[points.length - 1];
    ctx.lineTo(last.x, last.y);

    // Stroke line
    const accentColor = getComputedStyle(document.documentElement).getPropertyValue('--accent-color').trim() || '#fb923c';
    ctx.strokeStyle = accentColor;
    ctx.lineWidth = 2;
    ctx.stroke();

    // Fill gradient area
    ctx.lineTo(last.x, padding.top + chartH);
    ctx.lineTo(points[0].x, padding.top + chartH);
    ctx.closePath();

    const gradient = ctx.createLinearGradient(0, padding.top, 0, padding.top + chartH);
    gradient.addColorStop(0, accentColor + '40');
    gradient.addColorStop(0.5, accentColor + '15');
    gradient.addColorStop(1, accentColor + '00');
    ctx.fillStyle = gradient;
    ctx.fill();

    // Draw glow dot on last point with speed > 0
    if (this.aggregateSpeed > 0) {
      ctx.beginPath();
      ctx.arc(last.x, last.y, 4, 0, Math.PI * 2);
      ctx.fillStyle = accentColor;
      ctx.fill();
      ctx.beginPath();
      ctx.arc(last.x, last.y, 8, 0, Math.PI * 2);
      ctx.fillStyle = accentColor + '30';
      ctx.fill();
    }
  }

  formatBytes(bytes: number, decimals = 2) {
    if (!+bytes) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
  }

  formatSpeed(bytesPerSec: number): string {
    if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
    return this.formatBytes(bytesPerSec) + '/s';
  }

  formatETA(seconds: number): string {
    if (!seconds || seconds <= 0) return '--';
    if (seconds < 60) return `${Math.round(seconds)}s`;
    if (seconds < 3600) {
      const m = Math.floor(seconds / 60);
      const s = Math.round(seconds % 60);
      return `${m}m ${s}s`;
    }
    const h = Math.floor(seconds / 3600);
    const m = Math.round((seconds % 3600) / 60);
    return `${h}h ${m}m`;
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

  // Toast notification system — shows inline in topbar
  showToast(message: string, icon: string, color: string) {
    if (this.toastTimeout) clearTimeout(this.toastTimeout);
    this.currentToast = { message, icon, color, removing: false };
    this.cdr.detectChanges();
    this.toastTimeout = setTimeout(() => {
      if (this.currentToast) this.currentToast.removing = true;
      this.cdr.detectChanges();
      setTimeout(() => {
        this.currentToast = null;
        this.cdr.detectChanges();
      }, 250);
    }, 3000);
  }

  // Mode change handler (from sidebar)
  onModeChange(mode: string) {
    this.activeMode = mode;
    const themes: Record<string, { accent: string; glow: string }> = {
      'snail': { accent: '#facc15', glow: 'rgba(250, 204, 21, 0.15)' },
      'auto': { accent: '#fb923c', glow: 'rgba(251, 146, 60, 0.15)' },
      'turbo': { accent: '#4ade80', glow: 'rgba(74, 222, 128, 0.15)' }
    };
    const config = themes[mode];
    document.documentElement.style.setProperty('--accent-color', config.accent);
    document.documentElement.style.setProperty('--glow-color', config.glow);

    // Notify backend to adjust speed throttling
    this.http.post('http://localhost:8080/mode', { mode }).subscribe({
      next: () => console.log(`Mode set to ${mode}`),
      error: (err) => console.error('Failed to set mode:', err)
    });
  }

  // Drag & drop
  onDragOver(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = true;
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = false;
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = false;
    const text = event.dataTransfer?.getData('text/plain') || event.dataTransfer?.getData('text/uri-list') || '';
    if (text && (text.startsWith('http://') || text.startsWith('https://'))) {
      this.urlInput = text;
      this.openAddModal();
    }
  }
}