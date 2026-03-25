import { CommonModule } from '@angular/common';
import { Component, OnInit, signal } from '@angular/core';
import { DownloadManager } from './components/download-manager/download-manager';
import { Websocket } from './services/websocket';

@Component({
  selector: 'app-root',
  imports: [CommonModule,DownloadManager],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit{
  protected readonly title = signal('PullDown');

  constructor(private wsService: Websocket){}

  ngOnInit(): void {
    this.wsService.connect()
  }
}
