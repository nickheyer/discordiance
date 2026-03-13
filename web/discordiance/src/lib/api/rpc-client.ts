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
import { PlatformConfigService } from '$lib/proto/discordiance/v1/platform_config_pb';
import { AgentConfigService } from '$lib/proto/discordiance/v1/agent_config_pb';
import { ReporterConfigService } from '$lib/proto/discordiance/v1/reporter_config_pb';
import { PipelineService } from '$lib/proto/discordiance/v1/pipeline_pb';
import { IssueService } from '$lib/proto/discordiance/v1/issue_pb';
import { MessageService } from '$lib/proto/discordiance/v1/message_pb';
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
	public readonly platformConfig: Client<typeof PlatformConfigService>;
	public readonly agentConfig: Client<typeof AgentConfigService>;
	public readonly reporterConfig: Client<typeof ReporterConfigService>;
	public readonly pipeline: Client<typeof PipelineService>;
	public readonly issue: Client<typeof IssueService>;
	public readonly message: Client<typeof MessageService>;

	constructor() {
		this.health = createClient(HealthService, transport);
		this.product = createClient(ProductService, transport);
		this.platformConfig = createClient(PlatformConfigService, transport);
		this.agentConfig = createClient(AgentConfigService, transport);
		this.reporterConfig = createClient(ReporterConfigService, transport);
		this.pipeline = createClient(PipelineService, transport);
		this.issue = createClient(IssueService, transport);
		this.message = createClient(MessageService, transport);
	}
}

export const rpcClient = new RpcClient();
