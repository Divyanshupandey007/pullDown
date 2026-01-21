import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

export interface ProgressMessage{
  fileName: string
  percent: number
}

@Injectable({
  providedIn: 'root',
})
export class Websocket {
  private socket: WebSocket | undefined;
  
  // Create a "News Channel" for progress updates
  public progressUpdates$ = new Subject<ProgressMessage>();

  constructor() { }

  connect() {
    this.socket = new WebSocket('ws://localhost:8080/ws');

    this.socket.onopen = () => {
      console.log('✅ WebSocket Connected');
    };

    this.socket.onmessage = (event) => {
      // 1. Parse the JSON from Go
      const data = JSON.parse(event.data);
      
      // 2. If it is a progress event, publish it to the channel
      if (data.event === 'progress') {
        this.progressUpdates$.next({
          fileName: data.fileName,
          percent: data.percent
        });
      }
    };

    this.socket.onclose = () => console.log('❌ Disconnected');
  }
}
