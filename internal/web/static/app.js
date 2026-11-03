'use strict';

const $ = (selector) => document.querySelector(selector);
let allItems = [];
let toastTimer;

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  });
  if (response.status === 204) return null;
  const body = await response.json();
  if (!response.ok) throw new Error(body.error || 'Something went wrong');
  return body;
}

function stockStatus(item) {
  if (item.quantity === 0) return { key: 'out', text: 'Out of stock' };
  if (item.quantity <= item.min_stock) return { key: 'low', text: 'Low stock' };
  return { key: 'healthy', text: 'In stock' };
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = String(text);
  return node;
}

function makeCell(className) {
  return el('td', className);
}

function createItemRow(item) {
  const tr = el('tr');
  const info = makeCell('');
  const product = el('div', 'product-cell');
  product.appendChild(el('span', 'product-icon', item.category === 'Electronics' ? '⌘' : item.category === 'Office' ? '▤' : '⬡'));
  const names = el('div');
  names.append(el('strong', 'product-name', item.name), el('span', 'product-id', item.id));
  product.append(names);
  info.append(product);

  const category = makeCell('');
  category.append(el('span', 'category', item.category));
  const stock = makeCell('');
  stock.append(el('span', 'stock-number', item.quantity), el('span', 'stock-unit', ` ${item.unit}`));
  const state = stockStatus(item);
  const status = makeCell('');
  status.append(el('span', `status-badge status-${state.key}`, state.text));

  const actions = makeCell('actions-col');
  const actionWrap = el('div', 'quantity-actions');
  const minus = el('button', '', '−');
  minus.type = 'button';
  minus.setAttribute('aria-label', `Reduce ${item.name} quantity`);
  minus.disabled = item.quantity === 0;
  minus.addEventListener('click', () => changeQuantity(item, -1));
  const plus = el('button', '', '+');
  plus.type = 'button';
  plus.setAttribute('aria-label', `Increase ${item.name} quantity`);
  plus.addEventListener('click', () => changeQuantity(item, 1));
  const remove = el('button', 'delete-button', '×');
  remove.type = 'button';
  remove.setAttribute('aria-label', `Delete ${item.name}`);
  remove.addEventListener('click', () => deleteItem(item));
  actionWrap.append(minus, plus, remove);
  actions.append(actionWrap);
  tr.append(info, category, stock, status, actions);
  return tr;
}

function renderItems() {
  const search = $('#search-input').value.trim().toLowerCase();
  const category = $('#filter-category').value;
  const visible = allItems.filter((item) =>
    (category === 'all' || item.category === category) &&
    (item.name.toLowerCase().includes(search) || item.id.toLowerCase().includes(search))
  );
  const tbody = $('#items-body');
  tbody.replaceChildren(...visible.map(createItemRow));
  $('#empty-state').hidden = visible.length !== 0;
  $('#showing-count').textContent = `Showing ${visible.length} of ${allItems.length} products`;
  $('#item-count').textContent = allItems.length;
}

function renderCategories() {
  const selection = $('#filter-category');
  const current = selection.value;
  selection.replaceChildren(new Option('All categories', 'all'));
  [...new Set(allItems.map((item) => item.category))].sort().forEach((category) => {
    selection.add(new Option(category, category));
  });
  selection.value = [...selection.options].some((option) => option.value === current) ? current : 'all';
}

function renderAlerts() {
  const alerts = allItems.filter((item) => item.quantity <= item.min_stock)
    .sort((a, b) => a.quantity - b.quantity);
  const panel = $('#alert-list');
  $('#nav-alert-count').textContent = alerts.length;
  if (!alerts.length) {
    panel.replaceChildren(el('p', 'panel-lead', 'Great news — every product is above its minimum stock.'));
    return;
  }
  panel.replaceChildren(...alerts.map((item) => {
    const row = el('div', 'alert-item');
    row.append(el('span', 'alert-bullet', item.quantity === 0 ? '!' : '↘'));
    const details = el('div', 'alert-info');
    details.append(el('strong', '', item.name), el('small', '', `Minimum: ${item.min_stock} ${item.unit}`));
    row.append(details, el('span', item.quantity === 0 ? 'alert-amount zero' : 'alert-amount', `${item.quantity} left`));
    return row;
  }));
}

async function refresh() {
  const [items, summary] = await Promise.all([request('/api/items'), request('/api/summary')]);
  allItems = items;
  $('#stat-products').textContent = summary.total_items;
  $('#stat-units').textContent = summary.total_units;
  $('#stat-low').textContent = summary.low_stock;
  $('#stat-out').textContent = summary.out_of_stock;
  renderCategories();
  renderItems();
  renderAlerts();
}

function toast(message) {
  $('#toast').textContent = message;
  $('#toast').classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => $('#toast').classList.remove('show'), 2800);
}

async function changeQuantity(item, delta) {
  try {
    await request(`/api/items/${encodeURIComponent(item.id)}`, {
      method: 'PATCH', body: JSON.stringify({ quantity: item.quantity + delta }),
    });
    await refresh();
    toast(`${item.name} quantity updated`);
  } catch (error) { toast(error.message); }
}

async function deleteItem(item) {
  if (!confirm(`Remove "${item.name}" from inventory?`)) return;
  try {
    await request(`/api/items/${encodeURIComponent(item.id)}`, { method: 'DELETE' });
    await refresh();
    toast('Product removed');
  } catch (error) { toast(error.message); }
}

function setModal(visible) {
  $('#modal').hidden = !visible;
  $('#form-error').hidden = true;
  if (visible) $('#input-name').focus();
}

$('#today-date').textContent = new Date().toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
$('#search-input').addEventListener('input', renderItems);
$('#filter-category').addEventListener('change', renderItems);
$('#add-button').addEventListener('click', () => setModal(true));
$('#close-modal').addEventListener('click', () => setModal(false));
$('#cancel-modal').addEventListener('click', () => setModal(false));
$('#modal').addEventListener('click', (event) => { if (event.target === $('#modal')) setModal(false); });
document.addEventListener('keydown', (event) => { if (event.key === 'Escape') setModal(false); });
$('#item-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const values = Object.fromEntries(new FormData(event.target).entries());
  values.quantity = Number(values.quantity);
  values.min_stock = Number(values.min_stock);
  try {
    await request('/api/items', { method: 'POST', body: JSON.stringify(values) });
    event.target.reset();
    setModal(false);
    await refresh();
    toast('New product added successfully');
  } catch (error) {
    $('#form-error').textContent = error.message;
    $('#form-error').hidden = false;
  }
});

Promise.all([refresh(), request('/healthz')])
  .then(([, health]) => { $('#api-status').textContent = health.status === 'ok' ? 'Connected' : 'Unavailable'; })
  .catch((error) => { $('#api-status').textContent = 'Unavailable'; toast(`Unable to load inventory: ${error.message}`); });
