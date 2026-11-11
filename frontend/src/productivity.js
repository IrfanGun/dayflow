export const DEFAULT_SCORING_WEIGHTS = Object.freeze({ effort: 15, consistency: 20, output: 30, improvement: 35 });

const hasSession = result => Boolean(result.activity?.trim() || result.output?.trim() || result.summary?.trim() || Number(result.actualMinutes) > 0);
const dateInRange = (date, start, end) => date >= start && date <= end;
const keyFor = result => `${result.taskId}\u0000${result.date}`;
export const outputLines = result => (result.output || '').split(/\r?\n/).map(line => line.trim()).filter(Boolean);

function activityPoints(result) {
  if (!hasSession(result)) return 0;
  return 10 + Math.min(10, Math.floor(Math.max(0, Number(result.actualMinutes) || 0) / 30));
}

function buildHistory(tasks, results) {
  const taskByID = new Map(tasks.map(task => [task.id, task]));
  const uniqueResults = new Map();
  for (const result of results) uniqueResults.set(keyFor(result), result);
  const records = [...uniqueResults.values()]
    .map(result => ({ result, task: taskByID.get(result.taskId) }))
    .filter(record => record.task && hasSession(record.result))
    .sort((a, b) => a.result.date.localeCompare(b.result.date) || a.result.taskId.localeCompare(b.result.taskId));
  const previousOutputCountByTask = new Map();
  for (const record of records) {
    const lines = outputLines(record.result);
    const previousOutputCount = previousOutputCountByTask.get(record.task.id);
    record.outputImproved = lines.length > 0 && previousOutputCount !== undefined && lines.length > previousOutputCount;
    if (lines.length) previousOutputCountByTask.set(record.task.id, lines.length);
    record.activityPoints = activityPoints(record.result);
    const aiGrowthPoints = Number(record.result.aiGrowthPoints);
    if (Number.isInteger(aiGrowthPoints) && aiGrowthPoints >= 0 && aiGrowthPoints <= 100) {
      record.growthBreakdown = [{ points: aiGrowthPoints, reason: 'AI-reviewed growth based on recorded output and summary' }];
    } else {
      record.growthBreakdown = lines.map(reason => ({ points: 5, reason }));
      if (record.outputImproved) record.growthBreakdown.push({ points: 10, reason: 'More output items than the previous recorded session' });
    }
    record.growthPoints = record.growthBreakdown.reduce((sum, item) => sum + item.points, 0);
    record.activityPointBreakdown = { session: hasSession(record.result) ? 10 : 0, effort: Math.min(10, Math.floor(Math.max(0, Number(record.result.actualMinutes) || 0) / 30)) };
  }
  return records;
}

function ratioScore(current, previous) {
  if (!(previous > 0) || !Number.isFinite(current) || !Number.isFinite(previous)) return null;
  return Math.max(0, Math.min(100, 50 + 50 * (current / previous - 1)));
}

function outputTrendScores(current, previous) {
  const scores = [];
  const activityIDs = new Set([...current, ...previous].map(record => record.task.id));
  for (const activityID of activityIDs) {
    const currentOutputSessions = current.filter(record => record.task.id === activityID && outputLines(record.result).length > 0);
    const previousOutputSessions = previous.filter(record => record.task.id === activityID && outputLines(record.result).length > 0);
    if (!currentOutputSessions.length || !previousOutputSessions.length) continue;
    const currentRate = currentOutputSessions.reduce((sum, record) => sum + outputLines(record.result).length, 0) / currentOutputSessions.length;
    const previousRate = previousOutputSessions.reduce((sum, record) => sum + outputLines(record.result).length, 0) / previousOutputSessions.length;
    scores.push(ratioScore(currentRate, previousRate));
  }
  return scores.length ? scores.reduce((sum, score) => sum + score, 0) / scores.length : null;
}

function dateSpan(start, end) {
  return Math.max(1, Math.round((Date.parse(`${end}T12:00:00`) - Date.parse(`${start}T12:00:00`)) / 86400000) + 1);
}

function summarize(records, start, end, previousStart, previousEnd) {
  const current = records.filter(({ result }) => dateInRange(result.date, start, end));
  const previous = records.filter(({ result }) => dateInRange(result.date, previousStart, previousEnd));
  const activeDays = new Set(current.map(({ result }) => result.date));
  const previousActiveDays = new Set(previous.map(({ result }) => result.date));
  const outputCount = current.reduce((sum, record) => sum + outputLines(record.result).length, 0);
  const previousOutputCount = previous.reduce((sum, record) => sum + outputLines(record.result).length, 0);
  const effort = current.reduce((sum, { result }) => sum + Math.max(0, Number(result.actualMinutes) || 0), 0);
  const previousEffort = previous.reduce((sum, { result }) => sum + Math.max(0, Number(result.actualMinutes) || 0), 0);
  const dimensions = {
    effort: ratioScore(effort, previousEffort),
    consistency: ratioScore(activeDays.size, previousActiveDays.size),
    output: outputCount > 0 && previousOutputCount > 0 ? ratioScore(outputCount, previousOutputCount) : null,
    improvement: outputTrendScores(current, previous)
  };
  const complete = Object.values(dimensions).every(value => value !== null);
  const growthScore = complete
    ? Math.round(Object.entries(dimensions).reduce((sum, [key, value]) => sum + value * DEFAULT_SCORING_WEIGHTS[key], 0) / 100)
    : null;
  const streakData = [...activeDays].sort();
  let longestStreak = 0, streak = 0, priorDate = null;
  for (const date of streakData) {
    const day = Date.parse(`${date}T12:00:00`);
    streak = priorDate !== null && day - priorDate === 86400000 ? streak + 1 : 1;
    longestStreak = Math.max(longestStreak, streak);
    priorDate = day;
  }
  const today = new Date(); today.setHours(12, 0, 0, 0);
  const activeSet = new Set(streakData);
  let currentStreak = 0;
  for (let day = today; activeSet.has(`${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, '0')}-${String(day.getDate()).padStart(2, '0')}`); day.setDate(day.getDate() - 1)) currentStreak++;
  if (!currentStreak) {
    const yesterday = new Date(today); yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayKey = `${yesterday.getFullYear()}-${String(yesterday.getMonth() + 1).padStart(2, '0')}-${String(yesterday.getDate()).padStart(2, '0')}`;
    if (activeSet.has(yesterdayKey)) for (let day = yesterday; activeSet.has(`${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, '0')}-${String(day.getDate()).padStart(2, '0')}`); day.setDate(day.getDate() - 1)) currentStreak++;
  }
  const currentOutputRate = outputCount && current.length ? outputCount / current.length : null;
  const previousOutputRate = previousOutputCount && previous.length ? previousOutputCount / previous.length : null;
  const regularityTrend = dimensions.consistency === null ? 'Not enough history' : dimensions.consistency > 50 ? 'More active days than the previous period' : dimensions.consistency < 50 ? 'Fewer active days than the previous period' : 'Similar active days to the previous period';
  const outputTrend = dimensions.improvement === null ? 'Not enough comparable output history' : currentOutputRate > previousOutputRate ? 'More output items per session than the previous period' : currentOutputRate < previousOutputRate ? 'Fewer output items per session than the previous period' : 'Similar output items per session to the previous period';
  return {
    records: current, sessions: current.length, activeDays: activeDays.size,
    sessionsPerWeek: current.length / (dateSpan(start, end) / 7),
    longestStreak, currentStreak, effortMinutes: effort, previousEffortMinutes: previousEffort,
    previousActiveDays: previousActiveDays.size,
    activityPoints: current.reduce((sum, record) => sum + record.activityPoints, 0),
    outputCount, growthPoints: current.reduce((sum, record) => sum + record.growthPoints, 0),
    dimensions, growthScore, previousSessions: previous.length, previousOutputCount,
    regularityTrend, outputTrend
  };
}

export function computeProductivity({ tasks = [], results = [], start, end }) {
  const span = dateSpan(start, end);
  const previousEndDate = new Date(`${start}T12:00:00`); previousEndDate.setDate(previousEndDate.getDate() - 1);
  const previousStartDate = new Date(previousEndDate); previousStartDate.setDate(previousStartDate.getDate() - span + 1);
  const toDateKey = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
  const previousStart = toDateKey(previousStartDate), previousEnd = toDateKey(previousEndDate);
  const records = buildHistory(tasks, results);
  const summary = summarize(records, start, end, previousStart, previousEnd);
  const groups = new Map();
  for (const record of records) {
    const group = groups.get(record.task.id) || { task: record.task, records: [] };
    group.records.push(record); groups.set(record.task.id, group);
  }
  const activities = [...groups.values()].map(group => ({
    ...group,
    summary: summarize(group.records, start, end, previousStart, previousEnd),
    history: group.records
  })).filter(group => group.summary.sessions || group.history.some(({ result }) => dateInRange(result.date, previousStart, previousEnd)))
    .sort((a, b) => b.summary.effortMinutes - a.summary.effortMinutes || a.task.title.localeCompare(b.task.title));
  const categoryEffort = new Map();
  for (const record of summary.records) categoryEffort.set(record.task.category, (categoryEffort.get(record.task.category) || 0) + (Number(record.result.actualMinutes) || 0));
  return { ...summary, activities, categoryEffort: [...categoryEffort].map(([category, minutes]) => ({ category, minutes })).sort((a, b) => b.minutes - a.minutes), previousStart, previousEnd };
}

export function periodBounds(days, endDate = new Date()) {
  const end = new Date(endDate); end.setHours(12, 0, 0, 0);
  const start = new Date(end); start.setDate(start.getDate() - days + 1);
  const toKey = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
  return { start: toKey(start), end: toKey(end) };
}
