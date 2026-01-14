import { HttpClient } from '@angular/common/http';
import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-download-manager',
  imports: [FormsModule],
  templateUrl: './download-manager.html',
  styleUrl: './download-manager.css',
})
export class DownloadManager {
  url: string = ''; // Holds the input value

  constructor(private http: HttpClient) {}

  startDownload() {
    if (!this.url) {
      alert('Please enter a URL');
      return;
    }

    console.log("Sending to backend:", this.url);

    // Send POST request to Go Backend
    this.http.post('http://localhost:8080/download', { url: this.url })
      .subscribe({
        next: (res) => {
          console.log('Backend response:', res);
          alert('Download Started successfully!');
          this.url = ''; // Clear input
        },
        error: (err) => {
          console.error('Error:', err);
          alert('Failed to connect to Backend.');
        }
      });
  }
}
