import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';

import { CardListComponent } from './card-list.component';
import { CardListView } from '../../../models/content';
import { SharedModule } from '../../../shared.module';
import { IndicatorComponent } from '../indicator/indicator.component';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';

describe('CardListComponent', () => {
  let component: CardListComponent;
  let fixture: ComponentFixture<CardListComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      declarations: [IndicatorComponent],
      imports: [SharedModule, OverlayscrollbarsModule],
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CardListComponent);
    component = fixture.componentInstance;

    const view: CardListView = {
      config: { cards: [] },
      metadata: { type: 'cardsList' },
    };

    component.view = view;

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
