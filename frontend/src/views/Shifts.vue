<template>
  <div>
    <div class="toolbar">
      <div>
        <h2 class="page-title">工地排班</h2>
        <p class="page-sub">按工地与日期查看当日出勤安排</p>
      </div>
      <div style="display: flex; gap: 0.6rem;">
        <button class="btn secondary" @click="openPersonCreate">新增人员</button>
        <button class="btn" @click="openShiftCreate">新增排班</button>
      </div>
    </div>

    <div class="card" style="margin-bottom: 1rem;">
      <div class="filters">
        <label>
          工地
          <select v-model="filterSiteId" @change="loadShifts">
            <option v-for="s in sites" :key="s.id" :value="String(s.id)">{{ s.name }}</option>
          </select>
        </label>
        <label>
          日期
          <input v-model="filterDate" type="date" @change="loadShifts" />
        </label>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th>人员</th>
            <th>角色</th>
            <th>班次</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in shifts" :key="item.id">
            <td>{{ item.person?.displayName || '-' }}</td>
            <td>{{ item.person?.roleLabel || '-' }}</td>
            <td><span class="tag">{{ slotLabel(item.slot) }}</span></td>
            <td>
              <button class="btn secondary small" @click="openShiftEdit(item)">编辑</button>
              <button class="btn danger small" @click="removeShift(item)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="!shifts.length" class="page-sub">当日暂无排班</p>
      <p v-if="error" class="error">{{ error }}</p>
    </div>

    <div class="card">
      <h3 class="section-title">人员花名册</h3>
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
            <td><span class="tag" :class="{ off: !p.active }">{{ p.active ? '在岗' : '停用' }}</span></td>
            <td>
              <button class="btn secondary small" @click="openPersonEdit(p)">编辑</button>
              <button class="btn danger small" @click="removePerson(p)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="!persons.length" class="page-sub">暂无人员</p>
    </div>

    <div v-if="showShiftModal" class="modal-mask" @click.self="showShiftModal = false">
      <div class="modal">
        <h3>{{ shiftForm.id ? '编辑排班' : '新增排班' }}</h3>
        <div class="form-grid">
          <label>
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
            人员
            <select v-model.number="shiftForm.personId">
              <option :value="0" disabled>请选择</option>
              <option v-for="p in activePersons" :key="p.id" :value="p.id">
                {{ p.displayName }}（{{ p.roleLabel || '未设角色' }}）
              </option>
            </select>
          </label>
          <label>
            班次
            <select v-model="shiftForm.slot">
              <option value="morning">上午</option>
              <option value="afternoon">下午</option>
              <option value="full">全天</option>
            </select>
          </label>
        </div>
        <p v-if="shiftError" class="error">{{ shiftError }}</p>
        <div class="modal-actions">
          <button class="btn secondary" @click="showShiftModal = false">取消</button>
          <button class="btn" @click="saveShift">保存</button>
        </div>
      </div>
    </div>

    <div v-if="showPersonModal" class="modal-mask" @click.self="showPersonModal = false">
      <div class="modal">
        <h3>{{ personForm.id ? '编辑人员' : '新增人员' }}</h3>
        <div class="form-grid">
          <label>
            姓名
            <input v-model="personForm.displayName" placeholder="如 赵大力" />
          </label>
          <label>
            角色
            <input v-model="personForm.roleLabel" list="role-options" placeholder="如 技工 / 学生 / 监理" />
            <datalist id="role-options">
              <option value="技工" />
              <option value="学生" />
              <option value="监理" />
            </datalist>
          </label>
          <label v-if="personForm.id" class="full checkbox-label">
            <input v-model="personForm.active" type="checkbox" />
            在岗（取消勾选则停用，停用后不可排班）
          </label>
        </div>
        <p v-if="personError" class="error">{{ personError }}</p>
        <div class="modal-actions">
          <button class="btn secondary" @click="showPersonModal = false">取消</button>
          <button class="btn" @click="savePerson">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/http'

const route = useRoute()

const sites = ref([])
const persons = ref([])
const shifts = ref([])
const filterSiteId = ref('')
const filterDate = ref('')
const error = ref('')

const showShiftModal = ref(false)
const showPersonModal = ref(false)
const shiftError = ref('')
const personError = ref('')

const shiftForm = reactive({ id: null, siteId: 0, workDate: '', personId: 0, slot: 'morning' })
const personForm = reactive({ id: null, displayName: '', roleLabel: '', active: true })

const slotLabels = { morning: '上午', afternoon: '下午', full: '全天' }
const slotLabel = (s) => slotLabels[s] || s

const activePersons = computed(() => persons.value.filter((p) => p.active))

function todayStr() {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function loadSites() {
  const { data } = await api.get('/sites')
  sites.value = data
}

async function loadPersons() {
  const { data } = await api.get('/crew-persons')
  persons.value = data
}

async function loadShifts() {
  error.value = ''
  if (!filterSiteId.value || !filterDate.value) {
    shifts.value = []
    return
  }
  try {
    const { data } = await api.get('/crew-shifts', {
      params: { siteId: filterSiteId.value, from: filterDate.value, to: filterDate.value }
    })
    shifts.value = data
  } catch (e) {
    error.value = e.response?.data?.error || '加载失败'
  }
}

function openShiftCreate() {
  Object.assign(shiftForm, {
    id: null,
    siteId: Number(filterSiteId.value) || sites.value[0]?.id || 0,
    workDate: filterDate.value || todayStr(),
    personId: activePersons.value[0]?.id || 0,
    slot: 'morning'
  })
  shiftError.value = ''
  showShiftModal.value = true
}

function openShiftEdit(item) {
  Object.assign(shiftForm, {
    id: item.id,
    siteId: item.siteId,
    workDate: String(item.workDate).slice(0, 10),
    personId: item.personId,
    slot: item.slot
  })
  shiftError.value = ''
  showShiftModal.value = true
}

async function saveShift() {
  shiftError.value = ''
  try {
    const payload = {
      siteId: shiftForm.siteId,
      workDate: shiftForm.workDate,
      personId: shiftForm.personId,
      slot: shiftForm.slot
    }
    if (shiftForm.id) {
      await api.put(`/crew-shifts/${shiftForm.id}`, payload)
    } else {
      await api.post('/crew-shifts', payload)
    }
    showShiftModal.value = false
    await loadShifts()
  } catch (e) {
    shiftError.value = e.response?.data?.error || '保存失败'
  }
}

async function removeShift(item) {
  if (!confirm(`确认删除「${item.person?.displayName}」当日的排班？`)) return
  try {
    await api.delete(`/crew-shifts/${item.id}`)
    await loadShifts()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  }
}

function openPersonCreate() {
  Object.assign(personForm, { id: null, displayName: '', roleLabel: '', active: true })
  personError.value = ''
  showPersonModal.value = true
}

function openPersonEdit(p) {
  Object.assign(personForm, {
    id: p.id,
    displayName: p.displayName,
    roleLabel: p.roleLabel,
    active: p.active
  })
  personError.value = ''
  showPersonModal.value = true
}

async function savePerson() {
  personError.value = ''
  try {
    const payload = {
      displayName: personForm.displayName,
      roleLabel: personForm.roleLabel,
      active: personForm.active
    }
    if (personForm.id) {
      await api.put(`/crew-persons/${personForm.id}`, payload)
    } else {
      await api.post('/crew-persons', payload)
    }
    showPersonModal.value = false
    await loadPersons()
  } catch (e) {
    personError.value = e.response?.data?.error || '保存失败'
  }
}

async function removePerson(p) {
  if (!confirm(`确认删除人员「${p.displayName}」？`)) return
  try {
    await api.delete(`/crew-persons/${p.id}`)
    await loadPersons()
  } catch (e) {
    alert(e.response?.data?.error || '删除失败')
  }
}

onMounted(async () => {
  await loadSites()
  filterSiteId.value = String(route.query.siteId || sites.value[0]?.id || '')
  filterDate.value = todayStr()
  await Promise.all([loadPersons(), loadShifts()])
})
</script>

<style scoped>
.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.filters label {
  min-width: 220px;
}

.section-title {
  margin: 0 0 0.75rem;
  font-size: 1.05rem;
}

.tag.off {
  background: #e4ded6;
  color: var(--muted);
}

.checkbox-label {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}
</style>
