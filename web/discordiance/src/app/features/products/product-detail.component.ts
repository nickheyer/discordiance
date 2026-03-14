import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { TextareaModule } from 'primeng/textarea';
import { TagModule } from 'primeng/tag';
import { SelectModule } from 'primeng/select';
import { DialogModule } from 'primeng/dialog';
import { FormsModule } from '@angular/forms';
import { SlicePipe } from '@angular/common';
import { TransportService } from '../../core/services/transport.service';
import { ProductService } from '../../../gen/discordiance/v1/product_pb';
import { ProductContextTypeSchema } from '../../../gen/discordiance/v1/product_pb';
import type { Product, ProductContext } from '../../../gen/discordiance/v1/product_pb';

@Component({
  selector: 'app-product-detail',
  standalone: true,
  imports: [CardModule, ButtonModule, InputTextModule, TextareaModule, TagModule, SelectModule, DialogModule, FormsModule, SlicePipe],
  template: `
    @if (product) {
      <div class="page-header">
        <div>
          <div class="flex align-items-center gap-2 mb-1">
            <p-button icon="pi pi-arrow-left" [text]="true" severity="secondary" (onClick)="router.navigate(['/products'])" />
            <h1>{{ product.name }}</h1>
          </div>
          <p class="text-muted">{{ product.description }}</p>
        </div>
        <p-button label="Save" icon="pi pi-check" (onClick)="save()" />
      </div>

      <div class="grid">
        <div class="col-12 lg:col-6">
          <p-card header="Details">
            <div class="flex flex-column gap-3">
              <div class="flex flex-column gap-1">
                <label>Name</label>
                <input pInputText [(ngModel)]="product.name" />
              </div>
              <div class="flex flex-column gap-1">
                <label>Description</label>
                <textarea pTextarea [(ngModel)]="product.description" rows="4"></textarea>
              </div>
            </div>
          </p-card>
        </div>

        <div class="col-12 lg:col-6">
          <p-card header="Context">
            <div class="flex justify-content-end mb-3">
              <p-button label="Add Context" icon="pi pi-plus" size="small" (onClick)="showAddContext = true" />
            </div>
            @for (ctx of product.contexts; track ctx.id) {
              <div class="context-item">
                <div>
                  <p-tag [value]="contextTypeLabel(ctx.type)" severity="info" />
                  <span class="ml-2 font-semibold">{{ ctx.label }}</span>
                </div>
                <div class="flex align-items-center gap-2">
                  <span class="text-muted text-sm">{{ ctx.value | slice:0:60 }}</span>
                  <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small"
                    (onClick)="removeContext(ctx.id)" />
                </div>
              </div>
            }
            @if (product.contexts.length === 0) {
              <p class="text-muted">No context attached. Add files, URLs, or descriptions to improve classification.</p>
            }
          </p-card>
        </div>
      </div>

      <p-dialog header="Add Context" [(visible)]="showAddContext" [modal]="true" [style]="{width: '450px'}">
        <div class="flex flex-column gap-3 mt-2">
          <div class="flex flex-column gap-1">
            <label>Type</label>
            <p-select [(ngModel)]="newContextType" [options]="contextTypes" optionLabel="label" optionValue="value" placeholder="Select type" />
          </div>
          <div class="flex flex-column gap-1">
            <label>Label</label>
            <input pInputText [(ngModel)]="newContextLabel" placeholder="e.g. Main docs" />
          </div>
          <div class="flex flex-column gap-1">
            <label>Value</label>
            <textarea pTextarea [(ngModel)]="newContextValue" rows="3" placeholder="URL, text, or file path"></textarea>
          </div>
        </div>
        <ng-template #footer>
          <p-button label="Cancel" severity="secondary" [text]="true" (onClick)="showAddContext = false" />
          <p-button label="Add" icon="pi pi-check" (onClick)="addContext()" [disabled]="!newContextValue.trim()" />
        </ng-template>
      </p-dialog>
    }
  `,
  styles: `
    .page-header {
      display: flex; justify-content: space-between; align-items: flex-start;
      margin-bottom: 1.5rem;
      h1 { margin: 0; font-size: 1.75rem; font-weight: 700; }
      p { margin: 0.25rem 0 0 2.75rem; font-size: 0.9rem; }
    }
    .context-item {
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.6rem 0; border-bottom: 1px solid var(--p-surface-200);
      &:last-child { border-bottom: none; }
    }
  `,
})
export class ProductDetailComponent implements OnInit {
  private readonly client;
  product: Product | undefined;
  showAddContext = false;
  newContextType = 1;
  newContextLabel = '';
  newContextValue = '';

  contextTypes = [
    { label: 'Description', value: 1 },
    { label: 'URL', value: 2 },
    { label: 'Repository', value: 3 },
    { label: 'File', value: 4 },
    { label: 'Documentation', value: 5 },
  ];

  constructor(
    transport: TransportService,
    private route: ActivatedRoute,
    public router: Router,
  ) {
    this.client = transport.createClient(ProductService);
  }

  async ngOnInit() {
    const id = this.route.snapshot.paramMap.get('id')!;
    await this.load(id);
  }

  async load(id: string) {
    const res = await this.client.getProduct({ id });
    this.product = res.product;
  }

  async save() {
    if (!this.product) return;
    await this.client.updateProduct({
      id: this.product.id,
      name: this.product.name,
      description: this.product.description,
    });
  }

  async addContext() {
    if (!this.product) return;
    const res = await this.client.addProductContext({
      productId: this.product.id,
      type: this.newContextType,
      value: this.newContextValue.trim(),
      label: this.newContextLabel.trim(),
    });
    this.product = res.product;
    this.showAddContext = false;
    this.newContextLabel = '';
    this.newContextValue = '';
  }

  async removeContext(contextId: string) {
    if (!this.product) return;
    const res = await this.client.removeProductContext({
      productId: this.product.id,
      contextId,
    });
    this.product = res.product;
  }

  contextTypeLabel(type: number): string {
    return this.contextTypes.find((t) => t.value === type)?.label ?? 'Unknown';
  }
}
