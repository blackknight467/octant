import { Pipe, PipeTransform, SecurityContext } from '@angular/core';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';

@Pipe({
  name: 'escapepipe',
  standalone: false,
})
export class StringEscapePipe implements PipeTransform {
  constructor(private sanitizer: DomSanitizer) {}

  // sanitize() returns a plain string, so say so -- declaring SafeHtml here
  // made every downstream consumer look mistyped.
  public transform(value: string): string {
    return this.sanitizer.sanitize(
      SecurityContext.HTML,
      this.escapePipe(value)
    );
  }

  escapePipe(str: string): string {
    return str
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\n/g, '\\n');
  }
}
