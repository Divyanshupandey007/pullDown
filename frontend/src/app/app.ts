import { CommonModule } from '@angular/common';
import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { DownloadManager } from './components/download-manager/download-manager';
import { Websocket } from './services/websocket';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet,CommonModule,DownloadManager],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  protected readonly title = signal('PullDown');

  constructor(private wsService: Websocket){}

  ngOnInit(): void {
    this.wsService.connect()
  }
}
