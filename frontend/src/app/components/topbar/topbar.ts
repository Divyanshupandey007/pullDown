import { Component, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-topbar',
  imports: [CommonModule, FormsModule],
  templateUrl: './topbar.html',
  styleUrls: ['./topbar.css']
})
export class TopbarComponent {
  searchText: string = '';

  @Output() search = new EventEmitter<string>();
  @Output() addUrl = new EventEmitter<void>();
  @Output() resumeAll = new EventEmitter<void>();
  @Output() pauseAll = new EventEmitter<void>();
  @Output() stopAll = new EventEmitter<void>();
  
  // Placeholders for future
  @Output() openNotifications = new EventEmitter<void>();
  @Output() openSettings = new EventEmitter<void>();

  onSearchChange() {
    this.search.emit(this.searchText);
  }
}