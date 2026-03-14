import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { TextareaModule } from 'primeng/textarea';
import { SelectModule } from 'primeng/select';
import { MultiSelectModule } from 'primeng/multiselect';
import { TagModule } from 'primeng/tag';
import { TransportService } from '../../core/services/transport.service';
import { PipelineService } from '../../../gen/discordiance/v1/pipeline_pb';
import { ProductService } from '../../../gen/discordiance/v1/product_pb';
import { AgentService } from '../../../gen/discordiance/v1/agent_pb';
import { PlatformService } from '../../../gen/discordiance/v1/platform_pb';
import { ReporterService } from '../../../gen/discordiance/v1/reporter_pb';
import type { Pipeline } from '../../../gen/discordiance/v1/pipeline_pb';

interface SelectOption { label: string; value: string; }

@Component({
  selector: 'app-pipeline-detail',
  standalone: true,
  imports: [FormsModule, CardModule, ButtonModule, InputTextModule, TextareaModule, SelectModule, MultiSelectModule, TagModule],
  template: `
    <div class="page-header">
      <div class="flex align-items-center gap-2">
        <p-button icon="pi pi-arrow-left" [text]="true" severity="secondary" (onClick)="router.navigate(['/pipelines'])" />
        <h1>{{ isNew ? 'New Pipeline' : pipeline?.name }}</h1>
      </div>
      <p-button [label]="isNew ? 'Create' : 'Save'" icon="pi pi-check" (onClick)="save()" [disabled]="!name.trim() || !productId" />
    </div>

    <div class="grid">
      <div class="col-12 lg:col-6">
        <p-card header="Configuration">
          <div class="flex flex-column gap-3">
            <div class="flex flex-column gap-1">
              <label>Name</label>
              <input pInputText [(ngModel)]="name" placeholder="Pipeline name" />
            </div>
            <div class="flex flex-column gap-1">
              <label>Description</label>
              <textarea pTextarea [(ngModel)]="description" rows="2"></textarea>
            </div>
            <div class="flex flex-column gap-1">
              <label>Product</label>
              <p-select [(ngModel)]="productId" [options]="productOptions" optionLabel="label" optionValue="value" placeholder="Select product" />
            </div>
            <div class="flex flex-column gap-1">
              <label>Batch Mode</label>
              <p-select [(ngModel)]="batchMode" [options]="batchModes" optionLabel="label" optionValue="value" />
            </div>
          </div>
        </p-card>
      </div>

      <div class="col-12 lg:col-6">
        <p-card header="Components">
          <div class="flex flex-column gap-3">
            <div class="flex flex-column gap-1">
              <label>Agents</label>
              <p-multiselect [(ngModel)]="agentIds" [options]="agentOptions" optionLabel="label" optionValue="value" placeholder="Select agents" />
            </div>
            <div class="flex flex-column gap-1">
              <label>Platforms</label>
              <p-multiselect [(ngModel)]="platformIds" [options]="platformOptions" optionLabel="label" optionValue="value" placeholder="Select platforms" />
            </div>
            <div class="flex flex-column gap-1">
              <label>Reporters</label>
              <p-multiselect [(ngModel)]="reporterIds" [options]="reporterOptions" optionLabel="label" optionValue="value" placeholder="Select reporters" />
            </div>
          </div>
        </p-card>
      </div>
    </div>
  `,
  styles: `
    .page-header {
      display: flex; justify-content: space-between; align-items: center;
      margin-bottom: 1.5rem;
      h1 { margin: 0; font-size: 1.75rem; font-weight: 700; }
    }
  `,
})
export class PipelineDetailComponent implements OnInit {
  private readonly pipelineClient;
  private readonly productClient;
  private readonly agentClient;
  private readonly platformClient;
  private readonly reporterClient;

  pipeline: Pipeline | undefined;
  isNew = false;

  name = '';
  description = '';
  productId = '';
  agentIds: string[] = [];
  platformIds: string[] = [];
  reporterIds: string[] = [];
  batchMode = 1;

  productOptions: SelectOption[] = [];
  agentOptions: SelectOption[] = [];
  platformOptions: SelectOption[] = [];
  reporterOptions: SelectOption[] = [];

  batchModes = [
    { label: 'Single', value: 1 },
    { label: 'By Conversation', value: 2 },
    { label: 'By Author', value: 3 },
    { label: 'Time Window', value: 4 },
    { label: 'Fixed Size', value: 5 },
  ];

  constructor(
    transport: TransportService,
    private route: ActivatedRoute,
    public router: Router,
  ) {
    this.pipelineClient = transport.createClient(PipelineService);
    this.productClient = transport.createClient(ProductService);
    this.agentClient = transport.createClient(AgentService);
    this.platformClient = transport.createClient(PlatformService);
    this.reporterClient = transport.createClient(ReporterService);
  }

  async ngOnInit() {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.isNew = id === 'new';

    const [products, agents, platforms, reporters] = await Promise.all([
      this.productClient.listProducts({}),
      this.agentClient.listAgents({}),
      this.platformClient.listPlatforms({}),
      this.reporterClient.listReporters({}),
    ]);

    this.productOptions = products.products.map((p) => ({ label: p.name, value: p.id }));
    this.agentOptions = agents.agents.map((a) => ({ label: a.name, value: a.id }));
    this.platformOptions = platforms.platforms.map((p) => ({ label: p.name, value: p.id }));
    this.reporterOptions = reporters.reporters.map((r) => ({ label: r.name, value: r.id }));

    if (!this.isNew) {
      const res = await this.pipelineClient.getPipeline({ id });
      this.pipeline = res.pipeline;
      if (this.pipeline) {
        this.name = this.pipeline.name;
        this.description = this.pipeline.description;
        this.productId = this.pipeline.productId;
        this.agentIds = [...this.pipeline.agentIds];
        this.platformIds = [...this.pipeline.platformIds];
        this.reporterIds = [...this.pipeline.reporterIds];
        this.batchMode = this.pipeline.classificationStrategy?.batchMode ?? 1;
      }
    }
  }

  async save() {
    const payload = {
      name: this.name.trim(),
      description: this.description.trim(),
      productId: this.productId,
      agentIds: this.agentIds,
      platformIds: this.platformIds,
      reporterIds: this.reporterIds,
      classificationStrategy: { batchMode: this.batchMode, batchSize: 0, timeWindowSeconds: 0 },
    };

    if (this.isNew) {
      const res = await this.pipelineClient.createPipeline(payload);
      this.router.navigate(['/pipelines', res.pipeline?.id]);
    } else {
      await this.pipelineClient.updatePipeline({ id: this.pipeline!.id, ...payload });
    }
  }
}
