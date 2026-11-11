<script>
  import { onMount } from 'svelte';
  import ApexCharts from 'apexcharts';

  export let points = [];

  let chartHost;
  let chart;
  let chartReady = false;

  function makeOptions(values) {
    const ordered = [...values].sort((a, b) => a.date.localeCompare(b.date));
    let runningTotal = 0;
    const average = ordered.map((point, index) => {
      runningTotal += point.score;
      return Math.round(runningTotal / (index + 1));
    });
    return {
      series: [
        { name: 'Task score', type: 'column', data: ordered.map(point => point.score) },
        { name: 'Average trend', type: 'line', data: average }
      ],
      chart: {
        type: 'line', height: 250, toolbar: { show: false }, zoom: { enabled: false },
        animations: { enabled: true, easing: 'easeinout', speed: 450 },
        parentHeightOffset: 0, foreColor: '#8993a3'
      },
      colors: ['#b8a9ef', '#5473E6'],
      stroke: { curve: 'smooth', width: [0, 3], lineCap: 'round' },
      markers: { size: [0, 4], strokeWidth: 2, hover: { size: 6 } },
      fill: { opacity: [0.9, 1] },
      plotOptions: { bar: { columnWidth: '45%', borderRadius: 4 } },
      dataLabels: { enabled: false },
      legend: { show: true, position: 'top', horizontalAlign: 'right', fontSize: '10px', markers: { width: 8, height: 8, radius: 8 } },
      grid: { borderColor: '#edf0f5', strokeDashArray: 4, padding: { left: 2, right: 4, top: 0, bottom: 0 } },
      xaxis: {
        categories: ordered.map(point => point.label),
        axisBorder: { show: true, color: '#edf0f5' }, axisTicks: { show: false },
        labels: { style: { fontSize: '9px', colors: '#9aa3b0' }, trim: true, hideOverlappingLabels: true, rotate: -20 }
      },
      yaxis: {
        min: 0, max: 100, tickAmount: 4,
        labels: { formatter: value => `${Math.round(value)}`, style: { fontSize: '9px', colors: '#9aa3b0' } }
      },
      tooltip: { shared: true, intersect: false, y: { formatter: value => `${Math.round(value || 0)} / 100` }, style: { fontSize: '11px' } }
    };
  }

  onMount(() => {
    chart = new ApexCharts(chartHost, makeOptions(points));
    chart.render().then(() => chartReady = true);
    return () => chart?.destroy();
  });

  $: if (chart && chartReady) chart.updateOptions(makeOptions(points), true, true);
</script>

<div class="project-linearity-apex" bind:this={chartHost} role="img" aria-label="Task linearity scores as bars with a cumulative average trend line"></div>

<style>
  .project-linearity-apex { width: 100%; height: 250px; }
  :global(.project-linearity-apex .apexcharts-tooltip) { border: 1px solid #e7ebf2; border-radius: 9px; box-shadow: 0 8px 24px #27374b18; }
</style>
