import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { TextareaModule } from 'primeng/textarea';
import { FormsModule } from '@angular/forms';
import { SlicePipe } from '@angular/common';
import { TransportService } from '../../core/services/transport.service';
import { ProductService } from '../../../gen/discordiance/v1/product_pb';
import type { Product } from '../../../gen/discordiance/v1/product_pb';

@Component({
  selector: 'app-product-list',
  standalone: true,
  imports: [TableModule, ButtonModule, CardModule, DialogModule, InputTextModule, TextareaModule, FormsModule, SlicePipe],
  template: `
    <div class="page-header">
      <div>
        <h1>Products</h1>
        <p class="text-muted">Manage the subjects of your pipelines</p>
      </div>
      <p-button label="New Product" icon="pi pi-plus" (onClick)="showCreate = true" />
    </div>

    <p-card>
      <p-table [value]="products" [rowHover]="true" styleClass="p-datatable-sm">
        <ng-template #header>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Contexts</th>
            <th style="width: 5rem"></th>
          </tr>
        </ng-template>
        <ng-template #body let-product>
          <tr class="cursor-pointer" (click)="router.navigate(['/products', product.id])">
            <td><strong>{{ product.name }}</strong></td>
            <td class="text-muted">{{ product.description | slice:0:80 }}</td>
            <td>{{ product.contexts?.length ?? 0 }}</td>
            <td>
              <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small"
                (onClick)="deleteProduct($event, product.id)" />
            </td>
          </tr>
        </ng-template>
        <ng-template #emptymessage>
          <tr><td colspan="4" class="text-center text-muted p-4">No products yet. Create one to get started.</td></tr>
        </ng-template>
      </p-table>
    </p-card>

    <p-dialog header="New Product" [(visible)]="showCreate" [modal]="true" [style]="{width: '450px'}">
      <div class="flex flex-column gap-3 mt-2">
        <div class="flex flex-column gap-1">
          <label for="name">Name</label>
          <input pInputText id="name" [(ngModel)]="newName" placeholder="My Product" />
        </div>
        <div class="flex flex-column gap-1">
          <label for="desc">Description</label>
          <textarea pTextarea id="desc" [(ngModel)]="newDescription" rows="3" placeholder="What is this product?"></textarea>
        </div>
      </div>
      <ng-template #footer>
        <p-button label="Cancel" severity="secondary" [text]="true" (onClick)="showCreate = false" />
        <p-button label="Create" icon="pi pi-check" (onClick)="createProduct()" [disabled]="!newName.trim()" />
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
export class ProductListComponent implements OnInit {
  private readonly client;
  products: Product[] = [];
  showCreate = false;
  newName = '';
  newDescription = '';

  constructor(
    transport: TransportService,
    public router: Router,
    private cdr: ChangeDetectorRef,
  ) {
    this.client = transport.createClient(ProductService);
  }

  async ngOnInit() {
    await this.load();
  }

  async load() {
    const res = await this.client.listProducts({});
    this.products = res.products;
    this.cdr.markForCheck();
  }

  async createProduct() {
    await this.client.createProduct({
      name: this.newName.trim(),
      description: this.newDescription.trim(),
    });
    this.showCreate = false;
    this.newName = '';
    this.newDescription = '';
    await this.load();
  }

  async deleteProduct(event: Event, id: string) {
    event.stopPropagation();
    await this.client.deleteProduct({ id });
    await this.load();
  }
}
