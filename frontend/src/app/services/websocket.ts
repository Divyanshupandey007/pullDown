import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class Websocket {
  private socket: WebSocket | undefined;

  constructor() { }

  connect() {
    console.log('Attempting to connect to WebSocket...');
    
    // Connect to the Go WebSocket Route
    this.socket = new WebSocket('ws://localhost:8080/ws');

    // Event: Connection Established
    this.socket.onopen = () => {
      console.log('✅ WebSocket Connected to Go Backend');
    };

    // Event: Message Received from Go
    this.socket.onmessage = (event) => {
      console.log('📩 Message from Go:', event.data);
    };

    // Event: Connection Closed
    this.socket.onclose = (event) => {
      console.log('❌ WebSocket Disconnected', event);
    };

    // Event: Error
    this.socket.onerror = (error) => {
      console.error('⚠️ WebSocket Error:', error);
    };
  }
}
