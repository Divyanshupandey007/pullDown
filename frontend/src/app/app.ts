import { CommonModule } from '@angular/common';
import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { DownloadManager } from './components/download-manager/download-manager';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet,CommonModule,DownloadManager],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  protected readonly title = signal('PullDown');
}
