import {
	createClient,
	type Client,
	type Interceptor,
	ConnectError,
	type CallOptions
} from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { HealthService } from '$lib/proto/discordiance/v1/health_pb';
import { ProductService } from '$lib/proto/discordiance/v1/product_pb';
import { PlatformService } from '$lib/proto/discordiance/v1/platform_pb';
import { AgentService } from '$lib/proto/discordiance/v1/agent_pb';
import { ReporterService } from '$lib/proto/discordiance/v1/reporter_pb';
import { PipelineService } from '$lib/proto/discordiance/v1/pipeline_pb';
import { InsightService } from '$lib/proto/discordiance/v1/insight_pb';
import { MessageService } from '$lib/proto/discordiance/v1/message_pb';
import { ProductFileService } from '$lib/proto/discordiance/v1/file_pb';
import { toast } from 'svelte-sonner';

const errorInterceptor: Interceptor = (next) => async (req) => {
	try {
		return await next(req);
	} catch (err) {
		if (!req.header.get('X-Silent-Request')) {
			if (err instanceof ConnectError) {
				const message = err.rawMessage || err.message || 'An unexpected error occurred';
				toast.error(message);
			}
		}
		throw err;
	}
};

const transport = createConnectTransport({
	baseUrl: '',
	interceptors: [errorInterceptor]
});

export const silentCallOptions: CallOptions = {
	headers: new Headers({ 'X-Silent-Request': '1' })
};

export class RpcClient {
	public readonly health: Client<typeof HealthService>;
	public readonly product: Client<typeof ProductService>;
	public readonly platform: Client<typeof PlatformService>;
	public readonly agent: Client<typeof AgentService>;
	public readonly reporter: Client<typeof ReporterService>;
	public readonly pipeline: Client<typeof PipelineService>;
	public readonly insight: Client<typeof InsightService>;
	public readonly message: Client<typeof MessageService>;
	public readonly productFile: Client<typeof ProductFileService>;

	constructor() {
		this.health = createClient(HealthService, transport);
		this.product = createClient(ProductService, transport);
		this.platform = createClient(PlatformService, transport);
		this.agent = createClient(AgentService, transport);
		this.reporter = createClient(ReporterService, transport);
		this.pipeline = createClient(PipelineService, transport);
		this.insight = createClient(InsightService, transport);
		this.message = createClient(MessageService, transport);
		this.productFile = createClient(ProductFileService, transport);
	}
}

export const rpcClient = new RpcClient();
