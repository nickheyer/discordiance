import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'dashboard',
    pathMatch: 'full',
  },
  {
    path: 'dashboard',
    loadComponent: () =>
      import('./features/dashboard/dashboard.component').then(
        (m) => m.DashboardComponent
      ),
  },
  {
    path: 'products',
    loadComponent: () =>
      import('./features/products/product-list.component').then(
        (m) => m.ProductListComponent
      ),
  },
  {
    path: 'products/:id',
    loadComponent: () =>
      import('./features/products/product-detail.component').then(
        (m) => m.ProductDetailComponent
      ),
  },
  {
    path: 'pipelines',
    loadComponent: () =>
      import('./features/pipelines/pipeline-list.component').then(
        (m) => m.PipelineListComponent
      ),
  },
  {
    path: 'pipelines/:id',
    loadComponent: () =>
      import('./features/pipelines/pipeline-detail.component').then(
        (m) => m.PipelineDetailComponent
      ),
  },
  {
    path: 'insights',
    loadComponent: () =>
      import('./features/insights/insight-list.component').then(
        (m) => m.InsightListComponent
      ),
  },
  {
    path: 'settings',
    redirectTo: 'settings/agents',
    pathMatch: 'full',
  },
  {
    path: 'settings/agents',
    loadComponent: () =>
      import('./features/settings/agents/agent-list.component').then(
        (m) => m.AgentListComponent
      ),
  },
  {
    path: 'settings/platforms',
    loadComponent: () =>
      import('./features/settings/platforms/platform-list.component').then(
        (m) => m.PlatformListComponent
      ),
  },
  {
    path: 'settings/reporters',
    loadComponent: () =>
      import('./features/settings/reporters/reporter-list.component').then(
        (m) => m.ReporterListComponent
      ),
  },
];
