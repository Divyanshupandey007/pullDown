import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProgressMessage, Websocket } from '../../services/websocket';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-download-manager',
  imports: [CommonModule, FormsModule],
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager implements OnInit {
  url: string = '';
  progress: number = 0;
  currentFile: string = '';

  constructor(
    private http: HttpClient,
    private wsService: Websocket,
    private cdr: ChangeDetectorRef
  ) { }

  ngOnInit() {

    this.wsService.progressUpdates$.subscribe((msg: ProgressMessage) => {

      this.progress = msg.percent;
      this.currentFile = msg.fileName;

      // Explicitly trigger change detection for this component
      this.cdr.detectChanges();
    });
  }

  startDownload() {
    if (!this.url) return;
    this.progress = 0;

    this.http.post('http://localhost:8080/download', { url: this.url })
      .subscribe({
        next: (res) => console.log('Started'),
        error: (err) => alert('Error connecting to backend')
      });
  }
}
