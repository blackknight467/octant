import { TestBed, waitForAsync } from '@angular/core/testing';
import { ApplyYAMLComponent } from '../../components/smart/apply-yaml/apply-yaml.component';
import { FilterTextPipe } from './filtertext.pipe';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';

describe('FilterTextPipe', () => {
  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      imports: [OverlayscrollbarsModule],
      declarations: [ApplyYAMLComponent],
    });
  }));
  it('create an instance', () => {
    const pipe = new FilterTextPipe();
    expect(pipe).toBeTruthy();
  });
});
