import { Component, OnInit } from '@angular/core';
import { TableModule } from 'primeng/table';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';
import { SelectModule } from 'primeng/select';
import { FormsModule } from '@angular/forms';
import { DatePipe, SlicePipe } from '@angular/common';
import { TransportService } from '../../core/services/transport.service';
import { InsightService } from '../../../gen/discordiance/v1/insight_pb';
import type { Insight } from '../../../gen/discordiance/v1/insight_pb';

@Component({
  selector: 'app-insight-list',
  standalone: true,
  imports: [TableModule, CardModule, TagModule, SelectModule, FormsModule, DatePipe, SlicePipe],
  template: `
    <div class="page-header">
      <div>
        <h1>Insights</h1>
        <p class="text-muted">Browse and filter collected insights</p>
      </div>
    </div>

    <p-card>
      <div class="flex gap-3 mb-3">
        <p-select [(ngModel)]="stateFilter" [options]="stateOptions" optionLabel="label" optionValue="value"
          placeholder="Filter by state" [showClear]="true" (onChange)="load()" />
      </div>

      <p-table [value]="insights" [rowHover]="true" [paginator]="true" [rows]="25"
        [totalRecords]="totalCount" [lazy]="true" (onLazyLoad)="onPage($event)"
        styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th style="width: 6rem">State</th>
            <th>Content</th>
            <th>Author</th>
            <th>Medium</th>
            <th style="width: 10rem">Ingested</th>
          </tr>
        </ng-template>
        <ng-template #body let-insight>
          <tr>
            <td>
              <p-tag
                [severity]="stateSeverity(insight.state)"
                [value]="stateLabel(insight.state)"
              />
            </td>
            <td>{{ insight.content | slice:0:120 }}</td>
            <td>{{ insight.author }}</td>
            <td class="text-muted">{{ mediumLabel(insight.medium) }}</td>
            <td class="text-muted text-sm">{{ toDate(insight.ingestedAt) | date:'short' }}</td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="5" class="text-center text-muted p-4">No insights found</td></tr>
        </ng-template>
      </p-table>
    </p-card>
  `,
  styles: `
    .page-header {
      margin-bottom: 1.5rem;
      h1 { margin: 0 0 0.25rem; font-size: 1.75rem; font-weight: 700; }
      p { margin: 0; font-size: 0.9rem; }
    }
  `,
})
export class InsightListComponent implements OnInit {
  private readonly client;
  insights: Insight[] = [];
  totalCount = 0;
  stateFilter: number | null = null;

  stateOptions = [
    { label: 'Raw', value: 1 },
    { label: 'Cold', value: 2 },
    { label: 'Hot', value: 3 },
    { label: 'Burn', value: 4 },
  ];

  constructor(transport: TransportService) {
    this.client = transport.createClient(InsightService);
  }

  async ngOnInit() {
    await this.load();
  }

  async load(offset = 0) {
    const res = await this.client.listInsights({
      pagination: { pageSize: 25, pageToken: String(offset) },
      ...(this.stateFilter ? { state: this.stateFilter } : {}),
    });
    this.insights = res.insights;
    this.totalCount = res.pagination?.totalCount ?? 0;
  }

  async onPage(event: any) {
    await this.load(event.first ?? 0);
  }

  toDate(ts: any): Date | null {
    if (!ts) return null;
    if (ts.seconds != null) return new Date(Number(ts.seconds) * 1000);
    return null;
  }

  stateSeverity(s: number): 'danger' | 'warn' | 'info' | 'secondary' {
    switch (s) { case 4: return 'danger'; case 3: return 'warn'; case 2: return 'info'; default: return 'secondary'; }
  }

  stateLabel(s: number): string {
    switch (s) { case 1: return 'RAW'; case 2: return 'COLD'; case 3: return 'HOT'; case 4: return 'BURN'; default: return '?'; }
  }

  mediumLabel(m: number): string {
    switch (m) { case 1: return 'Post'; case 2: return 'Comment'; case 3: return 'Message'; case 4: return 'Thread'; case 5: return 'Review'; case 6: return 'Issue'; case 7: return 'Discussion'; default: return '—'; }
  }
}
