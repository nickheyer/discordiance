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
import { PlatformService } from '../../../../gen/discordiance/v1/platform_pb';
import type { Platform } from '../../../../gen/discordiance/v1/platform_pb';

@Component({
  selector: 'app-platform-list',
  standalone: true,
  imports: [FormsModule, TableModule, ButtonModule, CardModule, DialogModule, InputTextModule, SelectModule, TagModule],
  template: `
    <div class="page-header">
      <div>
        <h1>Platforms</h1>
        <p class="text-muted">Connect data sources for insight ingestion</p>
      </div>
      <p-button label="New Platform" icon="pi pi-plus" (onClick)="openCreate()" />
    </div>

    <p-card>
      <p-table [value]="platforms" [rowHover]="true" styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th style="width: 10rem"></th>
          </tr>
        </ng-template>
        <ng-template #body let-platform>
          <tr>
            <td><strong>{{ platform.name }}</strong></td>
            <td><p-tag [value]="typeLabel(platform.type)" severity="info" /></td>
            <td>
              <p-button icon="pi pi-bolt" [text]="true" size="small" pTooltip="Test Connection"
                (onClick)="testConnection(platform.id)" />
              <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small"
                (onClick)="remove(platform.id)" />
            </td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="3" class="text-center text-muted p-4">No platforms configured</td></tr>
        </ng-template>
      </p-table>
    </p-card>

    <p-dialog header="New Platform" [(visible)]="showDialog" [modal]="true" [style]="{width: '500px'}">
      <div class="flex flex-column gap-3 mt-2">
        <div class="flex flex-column gap-1">
          <label>Name</label>
          <input pInputText [(ngModel)]="form.name" placeholder="My Discord Server" />
        </div>
        <div class="flex flex-column gap-1">
          <label>Type</label>
          <p-select [(ngModel)]="form.type" [options]="typeOptions" optionLabel="label" optionValue="value" placeholder="Select type" />
        </div>

        @if (form.type === 1) {
          <div class="flex flex-column gap-1">
            <label>Bot Token</label>
            <input pInputText [(ngModel)]="form.discord.botToken" type="password" />
          </div>
          <div class="flex flex-column gap-1">
            <label>Guild IDs (comma-separated)</label>
            <input pInputText [(ngModel)]="form.discord.guildIds" />
          </div>
          <div class="flex flex-column gap-1">
            <label>Channel IDs (comma-separated)</label>
            <input pInputText [(ngModel)]="form.discord.channelIds" />
          </div>
        }
      </div>
      <ng-template #footer>
        <p-button label="Cancel" severity="secondary" [text]="true" (onClick)="showDialog = false" />
        <p-button label="Create" icon="pi pi-check" (onClick)="createPlatform()" [disabled]="!form.name.trim() || !form.type" />
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
export class PlatformListComponent implements OnInit {
  private readonly client;
  platforms: Platform[] = [];
  showDialog = false;
  form = this.emptyForm();

  typeOptions = [
    { label: 'Discord', value: 1 },
    { label: 'Reddit', value: 2 },
    { label: 'Twitter', value: 3 },
    { label: 'LinkedIn', value: 4 },
    { label: 'GitHub', value: 5 },
  ];

  constructor(transport: TransportService, private cdr: ChangeDetectorRef) {
    this.client = transport.createClient(PlatformService);
  }

  async ngOnInit() { await this.load(); }

  async load() {
    const res = await this.client.listPlatforms({});
    this.platforms = res.platforms;
    this.cdr.markForCheck();
  }

  openCreate() {
    this.form = this.emptyForm();
    this.showDialog = true;
  }

  async createPlatform() {
    const req: any = { name: this.form.name.trim(), type: this.form.type };

    if (this.form.type === 1) {
      req.discord = {
        botToken: this.form.discord.botToken,
        guildIds: this.form.discord.guildIds.split(',').map((s: string) => s.trim()).filter(Boolean),
        channelIds: this.form.discord.channelIds.split(',').map((s: string) => s.trim()).filter(Boolean),
      };
    }

    await this.client.createPlatform(req);
    this.showDialog = false;
    await this.load();
  }

  async testConnection(id: string) {
    const res = await this.client.testPlatformConnection({ id });
    this.cdr.markForCheck();
    alert(res.success ? 'Connection successful' : `Failed: ${res.message}`);
  }

  async remove(id: string) {
    await this.client.deletePlatform({ id });
    await this.load();
  }

  typeLabel(t: number): string {
    return this.typeOptions.find((o) => o.value === t)?.label ?? 'Unknown';
  }

  private emptyForm() {
    return { name: '', type: 0, discord: { botToken: '', guildIds: '', channelIds: '' } };
  }
}
