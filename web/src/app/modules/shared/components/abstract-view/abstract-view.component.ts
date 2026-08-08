import {
  AfterViewInit,
  EventEmitter,
  Input,
  OnInit,
  Output,
  Directive,
} from '@angular/core';
import { View } from '../../models/content';

@Directive()
// tslint:disable-next-line:directive-class-suffix
export abstract class AbstractViewComponent<T>
  implements OnInit, AfterViewInit
{
  v: T;

  @Input() set view(v: View) {
    this.v = v as unknown as T;
    this.update();
  }

  // Setter takes the base View (that is what the content system hands us);
  // getter returns the concrete T so templates can reach `view.config`.
  // Divergent accessor types are legal from TypeScript 4.3.
  get view(): T {
    return this.v;
  }

  @Output() viewInit: EventEmitter<void> = new EventEmitter<void>();

  private hasChildren = false;
  private initialChildCountActual = 0;

  protected set initialChildCount(n: number) {
    this.waitCountActual = n;
    if (n > 0) {
      this.hasChildren = true;
      this.initialChildCountActual = n;
    }
  }

  private waitCountActual = 0;
  protected set waitCount(n: number) {
    this.waitCountActual = n;
    if (n === 0 && this.hasChildren) {
      this.viewInit.emit();
      return;
    }
  }

  protected get waitCount(): number {
    return this.waitCountActual;
  }

  protected abstract update(): void;

  ngOnInit(): void {}

  ngAfterViewInit() {
    if (this.initialChildCountActual === 0) {
      this.viewInit.emit();
    }
  }

  ping() {
    this.update();
  }

  initDone() {
    this.waitCount = this.waitCount - 1;
  }
}
