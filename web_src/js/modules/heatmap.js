import ActivityHeatmap from '../components/ActivityHeatmap.svelte'
import { translateDay, translateMonth } from '../utils/index.js'

function initHeatmap() {
  const el = document.getElementById('user-heatmap')
  if (!el) return

  try {
    const heatmap = {}
    for (const { contributions, timestamp } of JSON.parse(
      el.getAttribute('data-heatmap-data')
    )) {
      // Convert to user timezone and sum contributions by date
      const dateStr = new Date(timestamp * 1000).toDateString()
      heatmap[dateStr] = (heatmap[dateStr] || 0) + contributions
    }

    const values = Object.keys(heatmap).map((v) => {
      return { date: new Date(v), count: heatmap[v] }
    })

    const locale = {
      months: new Array(12).fill().map((_, idx) => translateMonth(idx)),
      days: new Array(7).fill().map((_, idx) => translateDay(idx)),
      contributions: 'contributions',
      no_contributions: 'No contributions'
    }

    el.innerHTML = ''
    new ActivityHeatmap({
      target: el,
      props: { values, locale }
    })
  } catch (err) {
    console.error('Heatmap failed to load', err)
    el.textContent = 'Heatmap failed to load'
  }
}

initHeatmap()
