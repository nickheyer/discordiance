import { Component } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {
  navItems: NavItem[] = [
    { label: 'Dashboard', icon: 'pi pi-home', route: '/dashboard' },
    { label: 'Products', icon: 'pi pi-box', route: '/products' },
    { label: 'Pipelines', icon: 'pi pi-sitemap', route: '/pipelines' },
    { label: 'Insights', icon: 'pi pi-chart-line', route: '/insights' },
  ];

  settingsItems: NavItem[] = [
    { label: 'Agents', icon: 'pi pi-microchip-ai', route: '/settings/agents' },
    { label: 'Platforms', icon: 'pi pi-globe', route: '/settings/platforms' },
    { label: 'Reporters', icon: 'pi pi-megaphone', route: '/settings/reporters' },
  ];
}

interface NavItem {
  label: string;
  icon: string;
  route: string;
}
