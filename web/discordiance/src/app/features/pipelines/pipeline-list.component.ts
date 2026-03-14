import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';
import { TransportService } from '../../core/services/transport.service';
import { PipelineService } from '../../../gen/discordiance/v1/pipeline_pb';
import type { Pipeline } from '../../../gen/discordiance/v1/pipeline_pb';

@Component({
  selector: 'app-pipeline-list',
  standalone: true,
  imports: [TableModule, ButtonModule, CardModule, TagModule],
  template: `
    <div class="page-header">
      <div>
        <h1>Pipelines</h1>
        <p class="text-muted">Orchestrate your insight collection and classification</p>
      </div>
      <p-button label="New Pipeline" icon="pi pi-plus" (onClick)="router.navigate(['/pipelines', 'new'])" />
    </div>

    <p-card>
      <p-table [value]="pipelines" [rowHover]="true" styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Agents</th>
            <th>Platforms</th>
            <th>Reporters</th>
            <th style="width: 10rem">Actions</th>
          </tr>
        </ng-template>
        <ng-template #body let-pipeline>
          <tr class="cursor-pointer" (click)="router.navigate(['/pipelines', pipeline.id])">
            <td><strong>{{ pipeline.name }}</strong></td>
            <td>
              <p-tag
                [severity]="statusSeverity(pipeline.status)"
                [value]="statusLabel(pipeline.status)"
              />
            </td>
            <td>{{ pipeline.agentIds?.length ?? 0 }}</td>
            <td>{{ pipeline.platformIds?.length ?? 0 }}</td>
            <td>{{ pipeline.reporterIds?.length ?? 0 }}</td>
            <td>
              @if (pipeline.status !== 2) {
                <p-button icon="pi pi-play" severity="success" [text]="true" size="small"
                  (onClick)="start($event, pipeline.id)" pTooltip="Start" />
              } @else {
                <p-button icon="pi pi-pause" severity="warn" [text]="true" size="small"
                  (onClick)="pause($event, pipeline.id)" pTooltip="Pause" />
                <p-button icon="pi pi-stop" severity="danger" [text]="true" size="small"
                  (onClick)="stop($event, pipeline.id)" pTooltip="Stop" />
              }
              <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small"
                (onClick)="remove($event, pipeline.id)" />
            </td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="6" class="text-center text-muted p-4">No pipelines yet. Create one to start collecting insights.</td></tr>
        </ng-template>
      </p-table>
    </p-card>
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
export class PipelineListComponent implements OnInit {
  private readonly client;
  pipelines: Pipeline[] = [];

  constructor(transport: TransportService, public router: Router, private cdr: ChangeDetectorRef) {
    this.client = transport.createClient(PipelineService);
  }

  async ngOnInit() {
    await this.load();
  }

  async load() {
    const res = await this.client.listPipelines({});
    this.pipelines = res.pipelines;
    this.cdr.markForCheck();
  }

  async start(e: Event, id: string) { e.stopPropagation(); await this.client.startPipeline({ id }); await this.load(); }
  async pause(e: Event, id: string) { e.stopPropagation(); await this.client.pausePipeline({ id }); await this.load(); }
  async stop(e: Event, id: string) { e.stopPropagation(); await this.client.stopPipeline({ id }); await this.load(); }
  async remove(e: Event, id: string) { e.stopPropagation(); await this.client.deletePipeline({ id }); await this.load(); }

  statusSeverity(s: number): 'success' | 'warn' | 'danger' | 'secondary' {
    switch (s) { case 2: return 'success'; case 3: return 'warn'; case 4: return 'danger'; default: return 'secondary'; }
  }

  statusLabel(s: number): string {
    switch (s) { case 1: return 'IDLE'; case 2: return 'RUNNING'; case 3: return 'PAUSED'; case 4: return 'ERROR'; default: return 'UNKNOWN'; }
  }
}
