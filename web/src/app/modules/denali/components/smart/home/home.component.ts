import { Component, OnInit, ChangeDetectionStrategy } from '@angular/core';
import {
  darkTheme,
  ThemeService,
} from '../../../../shared/services/theme/theme.service';

@Component({
  selector: 'app-root',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class HomeComponent implements OnInit {
  constructor(private themeService: ThemeService) {}

  ngOnInit() {
    this.loadTheme();
  }

  loadTheme() {
    // TODO: enable theme switching for denali
    this.themeService.applyTheme(darkTheme.type);
  }
}
