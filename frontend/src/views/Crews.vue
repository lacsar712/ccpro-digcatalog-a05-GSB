<template>
  <div>
    <div class="toolbar">
      <div>
        <h2 class="page-title">工地排班</h2>
        <p class="page-sub">按工地与日期查看当日上工安排（上午 / 下午 / 全天）</p>
      </div>
      <div style="display: flex; gap: 0.5rem;">
        <button class="btn secondary" @click="openCreatePerson">新增人员</button>
        <button class="btn" @click="openCreateShift">新增排班</button>
      </div>
    </div>

    <div class="card">
      <div class="filter-row">
        <label>
          工地
          <select v-model="filterSiteId" @change="loadShifts">
            <option value="" disabled>请选择工地</option>
            <option v-for="s in sites" :key="s.id" :value="String(s.id)">{{ s.name }}</option>
          </select>
        </label>
        <label>
          日期
          <div class="date-picker">
            <button class="btn secondary small" type="button" @click="shiftDay(-1)">‹</button>
            <input v-model="workDate" type="date" @change="loadShifts" />
            <button class="btn secondary small" type="button" @click="shiftDay(1)">›</button>
          </div>
        </label>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>人员</th>
            <th>角色</th>
            <th>时段</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in shifts" :key="item.id">
            <td>{{ item.person?.displayName || '#' + item.personId }}</td>
            <td>{{ item.person?.roleLabel || '-' }}</td>
            <td><span class="tag">{{ slotLabel(item.slot) }}</span></td>
            <td>
              <button class="btn secondary small" @click="openEditShift(item)">编辑</button>
              <button class="btn danger small" @click="removeShift(item)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="filterSiteId && !shifts.length" class="page-sub">当日该工地暂无排班</p>
      <p v-if="shiftError" class="error">{{ shiftError }}</p>
    </div>

    <h3 class="section-title">人员名册</h3>
    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th>姓名</th>
            <th>角色</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in persons" :key="p.id">
            <td>{{ p.displayName }}</td>
            <td>{{ p.roleLabel || '-' }}</td>
            <td>
              <span class="tag" :class="p.active ? '' : 'tag-off'">{{ p.active ? '在岗' : '离岗' }}</span>
            </td>
            <td>
              <button class="btn secondary small" @click="openEditPerson(p)">编辑</button>
              <button class="btn danger small" @click="removePerson(p)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="!persons.length" class="page-sub">暂无人员</p>
      <p v-if="personError" class="error">{{ personError }}</p>
    </div>

    <!-- 排班弹窗 -->
    <div v-if="showShiftModal" class="modal-mask" @click.self="showShiftModal = false">
      <div class="modal">
        <h3>{{ shiftForm.id ? '编辑排班' : '新增排班' }}</h3>
        <div class="form-grid">
          <label class="full">
            工地
            <select v-model.number="shiftForm.siteId">
              <option :value="0" disabled>请选择</option>
              <option v-for="s in sites" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
          </label>
          <label>
            日期
            <input v-model="shiftForm.workDate" type="date" />
          </label>
          <label>
            时段
            <select v-model="shiftForm.slot">
              <option value="morning">上午</option>
              <option value="afternoon">下午</option>
              <option value="full">全天</option>
            </select>
          </label>
          <label class="full">
            人员
            <select v-model.number="shiftForm.personId">
              <option :value="0" disabled>请选择</option>
              <option v-for="p in persons" :key="p.id" :value="p.id">
                {{ p.displayName }}{{ p.roleLabel ? '（' + p.roleLabel + '）' : '' }}{{ p.active ? '' : '·离岗' }}
              </option>
            </select>
          </label>
        </div>
        <p v-if="shiftFormError" class="error">{{ shiftFormError }}</p>
        <div class="modal-actions">
          <button class="btn secondary" @click="showShiftModal = false">取消</button>
          <button class="btn" @click="saveShift">保存</button>
        </div>
      </div>
    </div>

    <!-- 人员弹窗 -->
    <div v-if="showPersonModal" class="modal-mask" @click.self="showPersonModal = false">
      <div class="modal">
        <h3>{{ personForm.id ? '编辑人员' : '新增人员' }}</h3>
        <div class="form-grid">
          <label class="full">
            姓名
            <input v-model="personForm.displayName" placeholder="如 韩铁柱" />
          </label>
          <label>
            角色
            <input v-model="personForm.roleLabel" list="role-options" placeholder="技工 / 学生 / 监理" />
            <datalist id="role-options">
              <option value="技工"></option>
              <option value="学生"></option>
              <option value="监理"></option>
            </datalist>
          </label>
          <label class="checkbox-label">
            <input v-model="personForm.active" type="checkbox" />
            在岗
          </label>
        </div>
        <p v-if="personFormError" class="error">{{ personFormError }}</p>
        <div class="modal-actions">
          <button class="btn secondary" @click="showPersonModal = false">取消</button>
          <button class="btn" @click="savePerson">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/http'

const route = useRoute()

const sites = ref([])
const persons = ref([])
const shifts = ref([])
const filterSiteId = ref('')
const workDate = ref(todayStr())

const shiftError = ref('')
const personError = ref('')
const shiftFormError = ref('')
const personFormError = ref('')
const showShiftModal = ref(false)
const showPersonModal = ref(false)

const shiftForm = reactive({ id: null, siteId: 0, workDate: '', personId: 0, slot: 'full' })
const personForm = reactive({ id: null, displayName: '', roleLabel: '', active: true })

const slotLabels = { morning: '上午', afternoon: '下午', full: '全天' }
function slotLabel(slot) {
  return slotLabels[slot] || slot
}

function todayStr() {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}

function shiftDay(delta) {
  const d = new Date(workDate.value + 'T00:00:00')
  d.setDate(d.getDate() + delta)
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  workDate.value = `${d.getFullYear()}-${m}-${day}`
  loadShifts()
}

async function loadSites() {
  const { data } = await api.get('/sites')
  sites.value = data
}

async function loadPersons() {
  personError.value = ''
  try {
    const { data } = await api.get('/crew-persons')
    persons.value = data
  } catch (e) {
    personError.value = e.response?.data?.error || '人员加载失败'
  }
}

async function loadShifts() {
  shiftError.value = ''
  if (!filterSiteId.value) {
    shifts.value = []
    return
  }
  try {
    const { data } = await api.get('/crew-shifts', {
      params: { siteId: filterSiteId.value, startDate: workDate.value, endDate: workDate.value }
    })
    shifts.value = data
  } catch (e) {
    shiftError.value = e.response?.data?.error || '排班加载失败'
  }
}

function openCreateShift() {
  Object.assign(shiftForm, {
    id: null,
    siteId: Number(filterSiteId.value) || sites.value[0]?.id || 0,
    workDate: workDate.value,
    personId: persons.value.find((p) => p.active)?.id || 0,
    slot: 'full'
  })
  shiftFormError.value = ''
  showShiftModal.value = true
}

function openEditShift(item) {
  Object.assign(shiftForm, {
    id: item.id,
    siteId: item.siteId,
    workDate: (item.workDate || '').slice(0, 10),
    personId: item.personId,
    slot: item.slot
  })
  shiftFormError.value = ''
  showShiftModal.value = true
}

async function saveShift() {
  shiftFormError.value = ''
  if (!shiftForm.siteId || !shiftForm.personId || !shiftForm.workDate) {
    shiftFormError.value = '工地、人员、日期必填'
    return
  }
  const payload = {
    siteId: shiftForm.siteId,
    personId: shiftForm.personId,
    workDate: shiftForm.workDate,
    slot: shiftForm.slot
  }
  try {
    if (shiftForm.id) {
      await api.put(`/crew-shifts/${shiftForm.id}`, payload)
    } else {
      await api.post('/crew-shifts', payload)
    }
    showShiftModal.value = false
    filterSiteId.value = String(shiftForm.siteId)
    workDate.value = shiftForm.workDate
    await loadShifts()
  } catch (e) {
    shiftFormError.value = e.response?.data?.error || '保存失败'
  }
}

async function removeShift(item) {
  const name = item.person?.displayName || `#${item.personId}`
  if (!confirm(`确认删除 ${item.workDate.slice(0, 10)} ${name} 的「${slotLabel(item.slot)}」排班？`)) return
  try {
    await api.delete(`/crew-shifts/${item.id}`)
    await loadShifts()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  }
}

function openCreatePerson() {
  Object.assign(personForm, { id: null, displayName: '', roleLabel: '', active: true })
  personFormError.value = ''
  showPersonModal.value = true
}

function openEditPerson(p) {
  Object.assign(personForm, { id: p.id, displayName: p.displayName, roleLabel: p.roleLabel, active: p.active })
  personFormError.value = ''
  showPersonModal.value = true
}

async function savePerson() {
  personFormError.value = ''
  if (!personForm.displayName.trim()) {
    personFormError.value = '姓名必填'
    return
  }
  const payload = {
    displayName: personForm.displayName.trim(),
    roleLabel: personForm.roleLabel.trim(),
    active: personForm.active
  }
  try {
    if (personForm.id) {
      await api.put(`/crew-persons/${personForm.id}`, payload)
    } else {
      await api.post('/crew-persons', payload)
    }
    showPersonModal.value = false
    await loadPersons()
  } catch (e) {
    personFormError.value = e.response?.data?.error || '保存失败'
  }
}

async function removePerson(p) {
  if (!confirm(`确认删除人员「${p.displayName}」？已有排班的人员无法删除，可改为离岗状态。`)) return
  try {
    await api.delete(`/crew-persons/${p.id}`)
    await loadPersons()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  }
}

onMounted(async () => {
  await loadSites()
  await loadPersons()
  const querySite = route.query.siteId ? String(route.query.siteId) : ''
  if (querySite && sites.value.some((s) => String(s.id) === querySite)) {
    filterSiteId.value = querySite
  } else {
    filterSiteId.value = sites.value[0] ? String(sites.value[0].id) : ''
  }
  if (route.query.date) {
    workDate.value = String(route.query.date).slice(0, 10)
  } else if (filterSiteId.value) {
    // 无显式日期时，若该工地已有排班则定位到最早的一个排班日
    try {
      const { data } = await api.get('/crew-shifts', { params: { siteId: filterSiteId.value } })
      if (data.length) workDate.value = data[0].workDate.slice(0, 10)
    } catch (e) {
      // 忽略，沿用今天
    }
  }
  await loadShifts()
})
</script>

<style scoped>
.filter-row {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.filter-row label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.88rem;
}

.filter-row select,
.date-picker input {
  min-width: 180px;
}

.date-picker {
  display: flex;
  gap: 0.35rem;
  align-items: center;
}

.date-picker .btn {
  padding: 0.3rem 0.7rem;
}

.section-title {
  margin: 1.5rem 0 0.75rem;
  font-size: 1.05rem;
}

.tag-off {
  background: #e9e4dc;
  color: #7a7063;
}

.checkbox-label {
  display: flex;
  flex-direction: row !important;
  align-items: center;
  gap: 0.4rem;
}
</style>
