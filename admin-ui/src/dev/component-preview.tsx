// Development-only entry; Vite's production build includes index.html alone.
import { createRoot } from 'react-dom/client';
import { Line, LineChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent } from '../components/ui/chart';
import { paperbackChartColors } from '../lib/chart-theme';
import { Button } from '../components/ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../components/ui/card';
import '../index.css';

const data = [{day:'Mon',logins:8,characters:2},{day:'Tue',logins:12,characters:3},{day:'Wed',logins:10,characters:1},{day:'Thu',logins:18,characters:5},{day:'Fri',logins:24,characters:4},{day:'Sat',logins:20,characters:6},{day:'Sun',logins:27,characters:3}];
createRoot(document.getElementById('root')!).render(<main className="mx-auto max-w-4xl space-y-6 p-8">
  <header><h1 className="text-3xl">Console component preview</h1><p className="mt-2 text-ink-muted">Development fixture. All chart values below are synthetic. This page is excluded from the production build.</p></header>
  <div className="flex gap-3"><Button>Primary action</Button><Button variant="outline">Secondary action</Button><Button disabled>Unavailable</Button></div>
  <Card><CardHeader><CardTitle className="text-xl">Weekly activity</CardTitle><CardDescription>Synthetic values for checking the chart foundation, labels and tooltip.</CardDescription></CardHeader><CardContent>
    <ChartContainer config={{logins:{label:'Logins',color:paperbackChartColors[0]},characters:{label:'New characters',color:paperbackChartColors[1]}}} className="h-80 w-full">
      <LineChart accessibilityLayer data={data} margin={{left:0,right:16,top:12,bottom:0}}><CartesianGrid vertical={false}/><XAxis dataKey="day" tickLine={false} axisLine={false}/><YAxis allowDecimals={false} tickLine={false} axisLine={false}/><ChartTooltip content={<ChartTooltipContent/>}/><ChartLegend content={<ChartLegendContent/>}/><Line dataKey="logins" stroke="var(--color-logins)" strokeWidth={2} dot={false} isAnimationActive={false}/><Line dataKey="characters" stroke="var(--color-characters)" strokeWidth={2} strokeDasharray="4 3" dot={false} isAnimationActive={false}/></LineChart>
    </ChartContainer>
  </CardContent></Card>
</main>);
