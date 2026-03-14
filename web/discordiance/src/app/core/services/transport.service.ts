import { Injectable } from '@angular/core';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createClient, type Client } from '@connectrpc/connect';
import type { DescService } from '@bufbuild/protobuf';

@Injectable({ providedIn: 'root' })
export class TransportService {
  private readonly transport = createConnectTransport({
    baseUrl: '',
  });

  createClient<T extends DescService>(service: T): Client<T> {
    return createClient(service, this.transport);
  }
}
