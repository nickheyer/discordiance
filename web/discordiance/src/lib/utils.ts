import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { timestampDate } from '@bufbuild/protobuf/wkt';

export function formatDate(ts?: Timestamp): string {
	if (!ts) return '—';
	try {
		const d = timestampDate(ts);
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	} catch {
		return '—';
	}
}

export function formatDateTime(ts?: Timestamp): string {
	if (!ts) return '—';
	try {
		const d = timestampDate(ts);
		return d.toLocaleString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	} catch {
		return '—';
	}
}

export function formatRelative(ts?: Timestamp): string {
	if (!ts) return '—';
	try {
		const d = timestampDate(ts);
		const now = Date.now();
		const diff = now - d.getTime();
		const seconds = Math.floor(diff / 1000);
		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		if (days < 30) return `${days}d ago`;
		return formatDate(ts);
	} catch {
		return '—';
	}
}

export function severityColor(severity: string): string {
	switch (severity.toLowerCase()) {
		case 'critical':
			return 'badge-destructive';
		case 'high':
			return 'badge-warning';
		case 'medium':
			return 'badge-info';
		case 'low':
			return 'badge-muted';
		default:
			return 'badge-muted';
	}
}

export function statusColor(status: string): string {
	switch (status.toLowerCase()) {
		case 'new':
			return 'badge-info';
		case 'confirmed':
		case 'open':
			return 'badge-warning';
		case 'resolved':
		case 'closed':
			return 'badge-success';
		case 'dismissed':
		case 'ignored':
			return 'badge-muted';
		default:
			return 'badge-muted';
	}
}

export function formatBytes(bytes: number | bigint): string {
	const b = Number(bytes);
	if (b === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB'];
	const i = Math.floor(Math.log(b) / Math.log(k));
	return parseFloat((b / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}
