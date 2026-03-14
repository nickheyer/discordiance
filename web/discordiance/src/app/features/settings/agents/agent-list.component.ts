import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { InputNumberModule } from 'primeng/inputnumber';
import { TransportService } from '../../../core/services/transport.service';
import { AgentService } from '../../../../gen/discordiance/v1/agent_pb';
import type { Agent } from '../../../../gen/discordiance/v1/agent_pb';

@Component({
  selector: 'app-agent-list',
  standalone: true,
  imports: [FormsModule, TableModule, ButtonModule, CardModule, DialogModule, InputTextModule, InputNumberModule],
  template: `
    <div class="page-header">
      <div>
        <h1>Agents</h1>
        <p class="text-muted">Configure LLM endpoints for insight classification</p>
      </div>
      <p-button label="New Agent" icon="pi pi-plus" (onClick)="openCreate()" />
    </div>

    <p-card>
      <p-table [value]="agents" [rowHover]="true" styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th>Name</th>
            <th>Model</th>
            <th>Endpoint</th>
            <th>API Key</th>
            <th style="width: 8rem"></th>
          </tr>
        </ng-template>
        <ng-template #body let-agent>
          <tr>
            <td><strong>{{ agent.name }}</strong></td>
            <td>{{ agent.model }}</td>
            <td class="text-muted">{{ agent.baseUrl }}</td>
            <td class="text-muted">{{ agent.apiKey }}</td>
            <td>
              <p-button icon="pi pi-pencil" [text]="true" size="small" (onClick)="openEdit(agent)" />
              <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small" (onClick)="remove(agent.id)" />
            </td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="5" class="text-center text-muted p-4">No agents configured</td></tr>
        </ng-template>
      </p-table>
    </p-card>

    <p-dialog [header]="editing ? 'Edit Agent' : 'New Agent'" [(visible)]="showDialog" [modal]="true" [style]="{width: '500px'}">
      <div class="flex flex-column gap-3 mt-2">
        <div class="flex flex-column gap-1">
          <label>Name</label>
          <input pInputText [(ngModel)]="form.name" placeholder="My Agent" />
        </div>
        <div class="flex flex-column gap-1">
          <label>Base URL</label>
          <input pInputText [(ngModel)]="form.baseUrl" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="flex flex-column gap-1">
          <label>Model</label>
          <input pInputText [(ngModel)]="form.model" placeholder="gpt-4o" />
        </div>
        <div class="flex flex-column gap-1">
          <label>API Key</label>
          <input pInputText [(ngModel)]="form.apiKey" type="password" placeholder="sk-..." />
        </div>
        <div class="grid">
          <div class="col-6 flex flex-column gap-1">
            <label>Max Tokens</label>
            <p-inputnumber [(ngModel)]="form.maxTokens" [min]="0" [max]="128000" />
          </div>
          <div class="col-6 flex flex-column gap-1">
            <label>Temperature</label>
            <p-inputnumber [(ngModel)]="form.temperature" [min]="0" [max]="2" [minFractionDigits]="1" [maxFractionDigits]="1" />
          </div>
        </div>
      </div>
      <ng-template #footer>
        <p-button label="Cancel" severity="secondary" [text]="true" (onClick)="showDialog = false" />
        <p-button [label]="editing ? 'Save' : 'Create'" icon="pi pi-check" (onClick)="saveAgent()" [disabled]="!form.name.trim() || !form.baseUrl.trim()" />
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
export class AgentListComponent implements OnInit {
  private readonly client;
  agents: Agent[] = [];
  showDialog = false;
  editing = false;
  form = this.emptyForm();

  constructor(transport: TransportService, private cdr: ChangeDetectorRef) {
    this.client = transport.createClient(AgentService);
  }

  async ngOnInit() { await this.load(); }

  async load() {
    const res = await this.client.listAgents({});
    this.agents = res.agents;
    this.cdr.markForCheck();
  }

  openCreate() {
    this.form = this.emptyForm();
    this.editing = false;
    this.showDialog = true;
  }

  openEdit(agent: Agent) {
    this.form = { id: agent.id, name: agent.name, baseUrl: agent.baseUrl, model: agent.model, apiKey: '', maxTokens: agent.maxTokens, temperature: agent.temperature };
    this.editing = true;
    this.showDialog = true;
  }

  async saveAgent() {
    if (this.editing) {
      await this.client.updateAgent({ id: this.form.id, name: this.form.name, baseUrl: this.form.baseUrl, model: this.form.model, apiKey: this.form.apiKey, maxTokens: this.form.maxTokens, temperature: this.form.temperature });
    } else {
      await this.client.createAgent({ name: this.form.name, baseUrl: this.form.baseUrl, model: this.form.model, apiKey: this.form.apiKey, maxTokens: this.form.maxTokens, temperature: this.form.temperature });
    }
    this.showDialog = false;
    await this.load();
  }

  async remove(id: string) {
    await this.client.deleteAgent({ id });
    await this.load();
  }

  private emptyForm() {
    return { id: '', name: '', baseUrl: '', model: '', apiKey: '', maxTokens: 4096, temperature: 0.3 };
  }
}
