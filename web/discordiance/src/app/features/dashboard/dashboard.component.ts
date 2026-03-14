import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';
import { TransportService } from '../../core/services/transport.service';
import { PipelineService } from '../../../gen/discordiance/v1/pipeline_pb';
import { InsightService } from '../../../gen/discordiance/v1/insight_pb';
import { ProductService } from '../../../gen/discordiance/v1/product_pb';
import type { Pipeline } from '../../../gen/discordiance/v1/pipeline_pb';
import type { InsightStats } from '../../../gen/discordiance/v1/insight_pb';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CardModule, TagModule],
  template: `
    <div class="page-header">
      <h1>Dashboard</h1>
      <p class="text-muted">Overview of your Discordiance engine</p>
    </div>

    <div class="grid">
      <div class="col-12 md:col-6 lg:col-3">
        <p-card>
          <div class="stat-card">
            <div class="stat-icon bg-blue-50 text-blue-600">
              <i class="pi pi-box"></i>
            </div>
            <div class="stat-content">
              <span class="stat-value">{{ productCount }}</span>
              <span class="stat-label">Products</span>
            </div>
          </div>
        </p-card>
      </div>

      <div class="col-12 md:col-6 lg:col-3">
        <p-card>
          <div class="stat-card">
            <div class="stat-icon bg-green-50 text-green-600">
              <i class="pi pi-sitemap"></i>
            </div>
            <div class="stat-content">
              <span class="stat-value">{{ runningPipelines }}</span>
              <span class="stat-label">Active Pipelines</span>
            </div>
          </div>
        </p-card>
      </div>

      <div class="col-12 md:col-6 lg:col-3">
        <p-card>
          <div class="stat-card">
            <div class="stat-icon bg-orange-50 text-orange-600">
              <i class="pi pi-chart-line"></i>
            </div>
            <div class="stat-content">
              <span class="stat-value">{{ stats?.total ?? 0 }}</span>
              <span class="stat-label">Total Insights</span>
            </div>
          </div>
        </p-card>
      </div>

      <div class="col-12 md:col-6 lg:col-3">
        <p-card>
          <div class="stat-card">
            <div class="stat-icon bg-red-50 text-red-600">
              <i class="pi pi-exclamation-triangle"></i>
            </div>
            <div class="stat-content">
              <span class="stat-value">{{ stats?.burnCount ?? 0 }}</span>
              <span class="stat-label">Burns</span>
            </div>
          </div>
        </p-card>
      </div>
    </div>

    <div class="grid mt-4">
      <div class="col-12 md:col-6">
        <p-card header="Insight Breakdown">
          <div class="breakdown-grid">
            <div class="breakdown-item">
              <p-tag severity="danger" value="BURN" />
              <span class="breakdown-value">{{ stats?.burnCount ?? 0 }}</span>
            </div>
            <div class="breakdown-item">
              <p-tag severity="warn" value="HOT" />
              <span class="breakdown-value">{{ stats?.hotCount ?? 0 }}</span>
            </div>
            <div class="breakdown-item">
              <p-tag severity="info" value="COLD" />
              <span class="breakdown-value">{{ stats?.coldCount ?? 0 }}</span>
            </div>
            <div class="breakdown-item">
              <p-tag severity="secondary" value="RAW" />
              <span class="breakdown-value">{{ stats?.rawCount ?? 0 }}</span>
            </div>
          </div>
        </p-card>
      </div>

      <div class="col-12 md:col-6">
        <p-card header="Pipeline Status">
          @for (pipeline of pipelines; track pipeline.id) {
            <div class="pipeline-row">
              <span class="pipeline-name">{{ pipeline.name }}</span>
              <p-tag
                [severity]="pipelineStatusSeverity(pipeline.status)"
                [value]="pipelineStatusLabel(pipeline.status)"
              />
            </div>
          }
          @if (pipelines.length === 0) {
            <p class="text-muted">No pipelines configured</p>
          }
        </p-card>
      </div>
    </div>
  `,
  styles: `
    .page-header {
      margin-bottom: 1.5rem;
      h1 { margin: 0 0 0.25rem; font-size: 1.75rem; font-weight: 700; }
      p { margin: 0; font-size: 0.9rem; }
    }
    .stat-card {
      display: flex; align-items: center; gap: 1rem;
    }
    .stat-icon {
      width: 3rem; height: 3rem; border-radius: 0.75rem;
      display: flex; align-items: center; justify-content: center;
      font-size: 1.25rem;
    }
    .stat-content {
      display: flex; flex-direction: column;
    }
    .stat-value {
      font-size: 1.5rem; font-weight: 700; line-height: 1;
    }
    .stat-label {
      font-size: 0.8rem; color: var(--p-text-muted-color); margin-top: 0.25rem;
    }
    .breakdown-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;
    }
    .breakdown-item {
      display: flex; align-items: center; justify-content: space-between;
      padding: 0.5rem 0;
    }
    .breakdown-value {
      font-size: 1.25rem; font-weight: 600;
    }
    .pipeline-row {
      display: flex; align-items: center; justify-content: space-between;
      padding: 0.6rem 0;
      border-bottom: 1px solid var(--p-surface-200);
      &:last-child { border-bottom: none; }
    }
    .pipeline-name {
      font-weight: 500;
    }
  `,
})
export class DashboardComponent implements OnInit {
  private readonly pipelineClient;
  private readonly insightClient;
  private readonly productClient;

  pipelines: Pipeline[] = [];
  stats: InsightStats | undefined;
  productCount = 0;
  runningPipelines = 0;

  constructor(transport: TransportService, private cdr: ChangeDetectorRef) {
    this.pipelineClient = transport.createClient(PipelineService);
    this.insightClient = transport.createClient(InsightService);
    this.productClient = transport.createClient(ProductService);
  }

  async ngOnInit() {
    const [pipelineRes, statsRes, productRes] = await Promise.all([
      this.pipelineClient.listPipelines({}),
      this.insightClient.getInsightStats({}),
      this.productClient.listProducts({}),
    ]);

    this.pipelines = pipelineRes.pipelines;
    this.stats = statsRes.stats;
    this.productCount = productRes.pagination?.totalCount ?? productRes.products.length;
    this.runningPipelines = this.pipelines.filter((p) => p.status === 2).length;
    this.cdr.markForCheck();
  }

  pipelineStatusSeverity(status: number): 'success' | 'warn' | 'danger' | 'info' | 'secondary' {
    switch (status) {
      case 2: return 'success';
      case 3: return 'warn';
      case 4: return 'danger';
      default: return 'secondary';
    }
  }

  pipelineStatusLabel(status: number): string {
    switch (status) {
      case 1: return 'IDLE';
      case 2: return 'RUNNING';
      case 3: return 'PAUSED';
      case 4: return 'ERROR';
      default: return 'UNKNOWN';
    }
  }
}
