import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

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
  public progressUpdates$ = new Subject<ProgressMessage>();

  connect() {
    this.socket = new WebSocket('ws://localhost:8080/ws');

    this.socket.onopen = () => console.log('✅ WS Connected');
    
    this.socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.event === 'progress') {
        this.progressUpdates$.next(data);
      }
    };
  }
}
