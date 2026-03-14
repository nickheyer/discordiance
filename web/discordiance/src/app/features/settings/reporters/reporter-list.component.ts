import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TagModule } from 'primeng/tag';
import { TransportService } from '../../../core/services/transport.service';
import { ReporterService } from '../../../../gen/discordiance/v1/reporter_pb';
import type { Reporter } from '../../../../gen/discordiance/v1/reporter_pb';

@Component({
  selector: 'app-reporter-list',
  standalone: true,
  imports: [FormsModule, TableModule, ButtonModule, CardModule, DialogModule, InputTextModule, SelectModule, TagModule],
  template: `
    <div class="page-header">
      <div>
        <h1>Reporters</h1>
        <p class="text-muted">Configure output destinations for classified insights</p>
      </div>
      <p-button label="New Reporter" icon="pi pi-plus" (onClick)="openCreate()" />
    </div>

    <p-card>
      <p-table [value]="reporters" [rowHover]="true" styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th style="width: 8rem"></th>
          </tr>
        </ng-template>
        <ng-template #body let-reporter>
          <tr>
            <td><strong>{{ reporter.name }}</strong></td>
            <td><p-tag [value]="typeLabel(reporter.type)" [severity]="reporter.type === 1 ? 'secondary' : 'info'" /></td>
            <td>
              @if (reporter.type !== 1) {
                <p-button icon="pi pi-bolt" [text]="true" size="small" pTooltip="Test"
                  (onClick)="testReporter(reporter.id)" />
                <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small"
                  (onClick)="remove(reporter.id)" />
              }
            </td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="3" class="text-center text-muted p-4">No reporters configured</td></tr>
        </ng-template>
      </p-table>
    </p-card>

    <p-dialog header="New Reporter" [(visible)]="showDialog" [modal]="true" [style]="{width: '500px'}">
      <div class="flex flex-column gap-3 mt-2">
        <div class="flex flex-column gap-1">
          <label>Name</label>
          <input pInputText [(ngModel)]="form.name" placeholder="My Webhook" />
        </div>
        <div class="flex flex-column gap-1">
          <label>Type</label>
          <p-select [(ngModel)]="form.type" [options]="typeOptions" optionLabel="label" optionValue="value" placeholder="Select type" />
        </div>

        @if (form.type === 2) {
          <div class="flex flex-column gap-1">
            <label>Webhook URL</label>
            <input pInputText [(ngModel)]="form.webhook.url" placeholder="https://..." />
          </div>
          <div class="flex flex-column gap-1">
            <label>Secret (optional)</label>
            <input pInputText [(ngModel)]="form.webhook.secret" type="password" />
          </div>
        }

        @if (form.type === 4) {
          <div class="flex flex-column gap-1">
            <label>Discord Webhook URL</label>
            <input pInputText [(ngModel)]="form.discord.webhookUrl" placeholder="https://discord.com/api/webhooks/..." />
          </div>
        }
      </div>
      <ng-template #footer>
        <p-button label="Cancel" severity="secondary" [text]="true" (onClick)="showDialog = false" />
        <p-button label="Create" icon="pi pi-check" (onClick)="createReporter()" [disabled]="!form.name.trim() || !form.type" />
      </ng-template>
    </p-dialog>
  `,
  styles: `
    .page-header {
      display: flex; justify-content: space-between; align-items: flex-start;
      margin-bottom: 1.5rem;
      h1 { margin: 0 0 0.25rem; font-size: 1.75rem; font-weight: 700; }
      p { margin: 0; font-size: 0.9rem; }
    }
  `,
})
export class ReporterListComponent implements OnInit {
  private readonly client;
  reporters: Reporter[] = [];
  showDialog = false;
  form = this.emptyForm();

  typeOptions = [
    { label: 'Webhook', value: 2 },
    { label: 'Email', value: 3 },
    { label: 'Discord', value: 4 },
    { label: 'GitHub Issue', value: 5 },
  ];

  constructor(transport: TransportService, private cdr: ChangeDetectorRef) {
    this.client = transport.createClient(ReporterService);
  }

  async ngOnInit() { await this.load(); }

  async load() {
    const res = await this.client.listReporters({});
    this.reporters = res.reporters;
    this.cdr.markForCheck();
  }

  openCreate() {
    this.form = this.emptyForm();
    this.showDialog = true;
  }

  async createReporter() {
    const req: any = { name: this.form.name.trim(), type: this.form.type };

    if (this.form.type === 2) {
      req.webhook = { url: this.form.webhook.url, secret: this.form.webhook.secret };
    } else if (this.form.type === 4) {
      req.discord = { webhookUrl: this.form.discord.webhookUrl };
    }

    await this.client.createReporter(req);
    this.showDialog = false;
    await this.load();
  }

  async testReporter(id: string) {
    const res = await this.client.testReporter({ id });
    this.cdr.markForCheck();
    alert(res.success ? 'Test successful' : `Failed: ${res.message}`);
  }

  async remove(id: string) {
    await this.client.deleteReporter({ id });
    await this.load();
  }

  typeLabel(t: number): string {
    switch (t) { case 1: return 'Internal'; case 2: return 'Webhook'; case 3: return 'Email'; case 4: return 'Discord'; case 5: return 'GitHub Issue'; default: return 'Unknown'; }
  }

  private emptyForm() {
    return { name: '', type: 0, webhook: { url: '', secret: '' }, discord: { webhookUrl: '' } };
  }
}
