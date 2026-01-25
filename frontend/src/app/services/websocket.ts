import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

// Match the Go "Task" struct
export interface Task {
  id: string;
  url: string;
  fileName: string;
  status: string;
  totalSize: number;
  downloaded: number;
  // We will calculate 'progress' in the component
  progress?: number; 
}

export interface ProgressMessage{
  event: string
  id: string
  fileName: string
  percent: number
}

@Injectable({
  providedIn: 'root',
})
export class Websocket {
   private socket: WebSocket | undefined;
  
  // Existing channel for live updates
  public progressUpdates$ = new Subject<ProgressMessage>();
  
  // NEW: Channel for loading history
  public historyUpdates$ = new Subject<Task[]>();

  constructor() {}

  connect() {
    this.socket = new WebSocket('ws://localhost:8080/ws');

    this.socket.onopen = () => console.log('✅ WS Connected');

    this.socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // 1. Handle Live Progress
        if (data.event === 'progress') {
          this.progressUpdates$.next(data);
        } 
        // 2. Handle Initial History (Load on Refresh)
        else if (data.event === 'initial_state') {
          // data.tasks comes from the Go backend
          this.historyUpdates$.next(data.tasks);
        }
      } catch (e) {
        console.error('WS Parse Error', e);
      }
    };

    this.socket.onclose = () => console.log('❌ Disconnected');
  }
}
