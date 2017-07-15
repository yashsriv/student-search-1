import {
  trigger,
  state,
  style,
  animate,
  transition
} from '@angular/animations';
import { Component, HostBinding, Input } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';
import { MdDialog } from '@angular/material';

import { DetailComponent } from '../detail';
import { SearchHelper } from '../../helpers/search.helper';
import { Student } from '../../models/student.model';

@Component({
  selector: 'search-student',
  templateUrl: './student.component.html',
  styleUrls: ['./student.component.css'],
  animations: [
    trigger('studentState', [
      state('*', style({opacity: 1})),
      transition(':enter', [
        style({opacity: 0}),
        animate('300ms ease-in', style({opacity: 1}))
      ]),
      transition(':leave', [
        style({opacity: 1}),
        animate('300ms ease-out', style({opacity: 0}))
      ]),
    ])
  ]
})
export class StudentComponent {
  @HostBinding('@studentState') fadeInAnimation = true;

  @Input()
  student: Student;

  parseYear = SearchHelper.ParseYear;

  constructor(private sanitizer: DomSanitizer,
              private dialog: MdDialog) {}

  get dept() {
    return SearchHelper.ParseBranch(this.student.d);
  }

  url = () => {
    return this.sanitizer.bypassSecurityTrustStyle(SearchHelper.ImageURL(this.student.g, this.student.i, this.student.u));
  }

  openDialog() {
    this.dialog.open(DetailComponent, {
      data: {
        student: this.student
      }
    });
  }

}
