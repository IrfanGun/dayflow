<script>
  import { onMount } from 'svelte';
  import ApexCharts from 'apexcharts';

  export let trend = [];
  export let mode = 'cumulative';

  let chartHost;
  let chart;
  let chartReady = false;

  function taskValues(points) {
    const totals = new Map();
    for (const point of points) {
      for (const segment of point.segments) {
        const current = totals.get(segment.task.id) || { task: segment.task, color: segment.color, growthPoints: 0 };
        current.growthPoints += segment.growthPoints;
        totals.set(segment.task.id, current);
      }
    }
    return [...totals.values()].filter(item => item.growthPoints > 0)
      .sort((a, b) => b.growthPoints - a.growthPoints || a.task.title.localeCompare(b.task.title));
  }

  function valuesFor(points, chartMode) {
    if (chartMode === 'tasks') return points.map(point => point.growthPoints);
    if (chartMode === 'cumulative') {
      let total = 0;
      return points.map(point => total += point.growthPoints);
    }

    return points.map(point => point.growthPoints);
  }

  function makeOptions(points, chartMode) {
    const cumulative = chartMode === 'cumulative';
    const taskData = chartMode === 'tasks' || cumulative ? taskValues(points) : [];
    const values = valuesFor(points, chartMode);
    const max = Math.max(1, ...values);
    const seriesName = cumulative ? 'Cumulative GP' : chartMode === 'average' ? 'Daily GP' : 'Growth Points';
    const taskSeries = taskData.map(item => ({
      name: item.task.title,
      type: 'column',
      data: points.map(point => point.segments.find(segment => segment.task.id === item.task.id)?.growthPoints || 0)
    }));
    const series = chartMode === 'tasks'
      ? taskData.map(item => ({
          name: item.task.title,
          data: points.map(point => point.segments.find(segment => segment.task.id === item.task.id)?.growthPoints || 0)
        }))
      : cumulative
        ? [...taskSeries, { name: 'Cumulative GP', type: 'line', data: values }]
      : [{ name: seriesName, data: values }];

    return {
      series,
      chart: {
        type: chartMode === 'tasks' ? 'bar' : 'line',
        height: chartMode === 'tasks' ? 270 : cumulative ? 270 : 205,
        stacked: chartMode === 'tasks' || cumulative,
        stackOnlyBar: true,
        toolbar: { show: false },
        zoom: { enabled: false },
        animations: { enabled: true, easing: 'easeinout', speed: 450 },
        parentHeightOffset: 0,
        foreColor: '#8993a3'
      },
      colors: chartMode === 'tasks' || cumulative ? [...taskData.map(item => item.color || item.task.color || '#5473E6'), ...(cumulative ? ['#5473E6'] : [])] : ['#8b73db'],
      stroke: chartMode === 'tasks' ? { width: 0 } : cumulative ? { curve: 'smooth', width: [...taskData.map(() => 0), 3], lineCap: 'round' } : { curve: 'smooth', width: 3, lineCap: 'round' },
      markers: { size: cumulative ? [...taskData.map(() => 0), 4] : chartMode === 'tasks' ? 0 : 2, strokeWidth: 2, hover: { size: 5 } },
      fill: { opacity: cumulative ? [...taskData.map(() => 0.88), 1] : 1 },
      dataLabels: {
        enabled: chartMode === 'tasks',
        formatter: value => value > 0 ? Math.round(value) : '',
        style: { fontSize: '9px', fontWeight: 700, colors: ['#fff'] },
        background: { enabled: false }
      },
      legend: { show: chartMode === 'tasks' || cumulative, position: 'bottom', horizontalAlign: 'center', fontSize: '10px', markers: { width: 8, height: 8, radius: 2 }, itemMargin: { horizontal: 8, vertical: 2 } },
      plotOptions: { bar: { horizontal: false, columnWidth: '58%', borderRadius: 3 } },
      grid: { borderColor: '#edf0f5', strokeDashArray: 4, padding: { left: 2, right: 2, top: 2, bottom: 0 } },
      xaxis: {
        categories: points.map(point => point.label),
        axisBorder: { show: true, color: '#edf0f5' },
        axisTicks: { show: false },
        labels: { style: { fontSize: '9px', colors: '#9aa3b0' }, trim: true, hideOverlappingLabels: true }
      },
      yaxis: cumulative ? [
        { min: 0, max: Math.max(1, ...points.map(point => point.growthPoints)), labels: { formatter: value => Math.round(value), style: { fontSize: '9px', colors: '#9aa3b0' } } },
        { seriesName: 'Cumulative GP', opposite: true, min: 0, max, labels: { formatter: value => Math.round(value), style: { fontSize: '9px', colors: '#9aa3b0' } } }
      ] : {
        min: 0,
        max,
        decimalsInFloat: chartMode === 'tasks' || cumulative ? 0 : 1,
        labels: { formatter: value => chartMode === 'tasks' || cumulative ? Math.round(value) : Number(value.toFixed(1)), style: { fontSize: '9px', colors: '#9aa3b0' } }
      },
      tooltip: {
        shared: cumulative,
        intersect: false,
        y: { formatter: value => `${cumulative || chartMode === 'tasks' ? Math.round(value) : value.toFixed(1)} GP` },
        style: { fontSize: '11px' }
      },
      noData: { text: 'No growth points in this period', style: { color: '#8993a3', fontSize: '11px' } }
    };
  }

  onMount(() => {
    chart = new ApexCharts(chartHost, makeOptions(trend, mode));
    chart.render().then(() => chartReady = true);
    return () => chart?.destroy();
  });

  $: if (chart && chartReady) chart.updateOptions(makeOptions(trend, mode), true, true);
</script>

<div class="productivity-apex-chart" style:height={mode === 'tasks' ? '270px' : mode === 'cumulative' ? '270px' : '205px'} bind:this={chartHost} role="img" aria-label={mode === 'cumulative' ? 'Daily task growth bars and cumulative growth line' : mode === 'average' ? 'Growth points earned each day' : 'Daily growth points by task'}></div>

<style>
  .productivity-apex-chart { width: 100%; min-height: 205px; margin-top: 8px; }
  :global(.productivity-apex-chart .apexcharts-tooltip) { border: 1px solid #e7ebf2; border-radius: 9px; box-shadow: 0 8px 24px #27374b18; }
</style>
