import { Pipe, PipeTransform } from '@angular/core';
import {
  DomSanitizer,
  SafeHtml,
  SafeResourceUrl,
  SafeScript,
  SafeStyle,
  SafeUrl,
} from '@angular/platform-browser';
import { default as AnsiUp } from 'ansi_up';

@Pipe({
  name: 'ansipipe',
  standalone: false,
})
export class AnsiPipe implements PipeTransform {
  ansiUp = new AnsiUp();

  constructor(private sanitizer: DomSanitizer) {}

  // Only the ANSI branch produces SafeHtml; the common path returns the input
  // string unchanged.
  public transform(value: string | SafeHtml): SafeHtml | string {
    // May already be SafeHtml if an earlier step in the chain sanitised it.
    if (typeof value !== 'string') {
      return value;
    }
    if (value.includes('\x1B')) {
      // ANSI string
      return this.sanitizer.bypassSecurityTrustHtml(
        this.ansiUp.ansi_to_html(value)
      );
    }
    return value;
  }
}
