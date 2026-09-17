package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"digcatalog/internal/middleware"
	"digcatalog/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	DB        *gorm.DB
	JWTSecret string
}

func New(db *gorm.DB, jwtSecret string) *Handler {
	return &Handler{DB: db, JWTSecret: jwtSecret}
}

// isDuplicateErr 识别 MySQL 唯一键冲突（Error 1062 Duplicate entry）
func isDuplicateErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Error 1062")
}

// ---------- Auth ----------

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}
	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := middleware.GenerateToken(h.JWTSecret, user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *Handler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id":       c.MustGet("userId"),
		"username": c.MustGet("username"),
		"role":     c.MustGet("role"),
	})
}

// ---------- Sites ----------

func (h *Handler) ListSites(c *gin.Context) {
	var sites []models.Site
	if err := h.DB.Order("id desc").Find(&sites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sites)
}

func (h *Handler) GetSite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var site models.Site
	if err := h.DB.First(&site, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工地不存在"})
		return
	}
	c.JSON(http.StatusOK, site)
}

func (h *Handler) CreateSite(c *gin.Context) {
	var site models.Site
	if err := c.ShouldBindJSON(&site); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if site.Name == "" || site.Period == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称和时代必填"})
		return
	}
	if err := h.DB.Create(&site).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, site)
}

func (h *Handler) UpdateSite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var site models.Site
	if err := h.DB.First(&site, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工地不存在"})
		return
	}
	var req models.Site
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	site.Name = req.Name
	site.Period = req.Period
	site.Latitude = req.Latitude
	site.Longitude = req.Longitude
	site.Manager = req.Manager
	if err := h.DB.Save(&site).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, site)
}

func (h *Handler) DeleteSite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var count int64
	h.DB.Model(&models.Unit{}).Where("site_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该工地下仍有探方，无法删除"})
		return
	}
	if err := h.DB.Delete(&models.Site{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Units ----------

func (h *Handler) ListUnits(c *gin.Context) {
	var units []models.Unit
	q := h.DB.Preload("Site").Order("id desc")
	if siteID := c.Query("siteId"); siteID != "" {
		q = q.Where("site_id = ?", siteID)
	}
	if err := q.Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, units)
}

func (h *Handler) GetUnit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var unit models.Unit
	if err := h.DB.Preload("Site").First(&unit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "探方不存在"})
		return
	}
	c.JSON(http.StatusOK, unit)
}

func (h *Handler) CreateUnit(c *gin.Context) {
	var unit models.Unit
	if err := c.ShouldBindJSON(&unit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if unit.SiteID == 0 || unit.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属工地和编号必填"})
		return
	}
	var site models.Site
	if err := h.DB.First(&site, unit.SiteID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属工地不存在"})
		return
	}
	if err := h.DB.Create(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Site").First(&unit, unit.ID)
	c.JSON(http.StatusCreated, unit)
}

func (h *Handler) UpdateUnit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var unit models.Unit
	if err := h.DB.First(&unit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "探方不存在"})
		return
	}
	var req models.Unit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	unit.SiteID = req.SiteID
	unit.Code = req.Code
	unit.DepthMin = req.DepthMin
	unit.DepthMax = req.DepthMax
	unit.StratumDesc = req.StratumDesc
	if err := h.DB.Save(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Site").First(&unit, unit.ID)
	c.JSON(http.StatusOK, unit)
}

func (h *Handler) DeleteUnit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var count int64
	h.DB.Model(&models.Find{}).Where("unit_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该探方下仍有文物，无法删除"})
		return
	}
	if err := h.DB.Delete(&models.Unit{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Materials ----------

func (h *Handler) ListMaterials(c *gin.Context) {
	var materials []models.Material
	if err := h.DB.Order("id asc").Find(&materials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, materials)
}

func (h *Handler) CreateMaterial(c *gin.Context) {
	var m models.Material
	if err := c.ShouldBindJSON(&m); err != nil || m.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称必填"})
		return
	}
	if err := h.DB.Create(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) UpdateMaterial(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var m models.Material
	if err := h.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "材质不存在"})
		return
	}
	var req models.Material
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	m.Name = req.Name
	m.Description = req.Description
	if err := h.DB.Save(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) DeleteMaterial(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(&models.Material{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Finds ----------

type findReq struct {
	UnitID       uint    `json:"unitId"`
	MaterialID   *uint   `json:"materialId"`
	RegisterNo   string  `json:"registerNo"`
	ArtifactType string  `json:"artifactType"`
	MaterialName string  `json:"materialName"`
	Completeness string  `json:"completeness"`
	FindDate     *string `json:"findDate"`
	Description  string  `json:"description"`
	StorageLoc   string  `json:"storageLoc"`
}

func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func (h *Handler) ListFinds(c *gin.Context) {
	var finds []models.Find
	q := h.DB.Preload("Unit").Preload("Unit.Site").Preload("Material").Order("id desc")
	if unitID := c.Query("unitId"); unitID != "" {
		q = q.Where("unit_id = ?", unitID)
	}
	if at := c.Query("artifactType"); at != "" {
		q = q.Where("artifact_type = ?", at)
	}
	if err := q.Find(&finds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, finds)
}

func (h *Handler) GetFind(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var find models.Find
	if err := h.DB.Preload("Unit").Preload("Material").First(&find, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文物不存在"})
		return
	}
	c.JSON(http.StatusOK, find)
}

func (h *Handler) applyFindReq(find *models.Find, req *findReq) {
	find.UnitID = req.UnitID
	find.MaterialID = req.MaterialID
	find.RegisterNo = req.RegisterNo
	find.ArtifactType = req.ArtifactType
	find.MaterialName = req.MaterialName
	find.Completeness = req.Completeness
	find.FindDate = parseDate(req.FindDate)
	find.Description = req.Description
	find.StorageLoc = req.StorageLoc
	if find.MaterialID != nil {
		var m models.Material
		if err := h.DB.First(&m, *find.MaterialID).Error; err == nil {
			find.MaterialName = m.Name
		}
	}
}

func (h *Handler) CreateFind(c *gin.Context) {
	var req findReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if req.UnitID == 0 || req.RegisterNo == "" || req.ArtifactType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "探方、登记号、器物类型必填"})
		return
	}
	var unit models.Unit
	if err := h.DB.First(&unit, req.UnitID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属探方不存在"})
		return
	}
	var find models.Find
	h.applyFindReq(&find, &req)
	if err := h.DB.Create(&find).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Unit").Preload("Material").First(&find, find.ID)
	c.JSON(http.StatusCreated, find)
}

func (h *Handler) UpdateFind(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var find models.Find
	if err := h.DB.First(&find, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文物不存在"})
		return
	}
	var req findReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	h.applyFindReq(&find, &req)
	if err := h.DB.Save(&find).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Unit").Preload("Material").First(&find, find.ID)
	c.JSON(http.StatusOK, find)
}

func (h *Handler) DeleteFind(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(&models.Find{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Crew persons ----------

func (h *Handler) ListCrewPersons(c *gin.Context) {
	var persons []models.CrewPerson
	q := h.DB.Order("id desc")
	if active := c.Query("active"); active != "" {
		q = q.Where("active = ?", active == "true" || active == "1")
	}
	if err := q.Find(&persons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, persons)
}

func (h *Handler) GetCrewPerson(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var person models.CrewPerson
	if err := h.DB.First(&person, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人员不存在"})
		return
	}
	c.JSON(http.StatusOK, person)
}

type crewPersonReq struct {
	DisplayName string `json:"displayName"`
	RoleLabel   string `json:"roleLabel"`
	Active      *bool  `json:"active"`
}

func (h *Handler) CreateCrewPerson(c *gin.Context) {
	var req crewPersonReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "姓名必填"})
		return
	}
	person := models.CrewPerson{
		DisplayName: strings.TrimSpace(req.DisplayName),
		RoleLabel:   strings.TrimSpace(req.RoleLabel),
		Active:      true,
	}
	if req.Active != nil {
		person.Active = *req.Active
	}
	if err := h.DB.Select("display_name", "role_label", "active").Create(&person).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, person)
}

func (h *Handler) UpdateCrewPerson(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var person models.CrewPerson
	if err := h.DB.First(&person, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人员不存在"})
		return
	}
	var req crewPersonReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "姓名必填"})
		return
	}
	person.DisplayName = strings.TrimSpace(req.DisplayName)
	person.RoleLabel = strings.TrimSpace(req.RoleLabel)
	if req.Active != nil {
		person.Active = *req.Active
	}
	if err := h.DB.Save(&person).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, person)
}

func (h *Handler) DeleteCrewPerson(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var count int64
	h.DB.Model(&models.CrewShift{}).Where("person_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该人员仍有排班记录，无法删除"})
		return
	}
	if err := h.DB.Delete(&models.CrewPerson{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Crew shifts ----------

var crewSlots = map[string]bool{"morning": true, "afternoon": true, "full": true}

type crewShiftReq struct {
	SiteID   uint   `json:"siteId"`
	WorkDate string `json:"workDate"`
	PersonID uint   `json:"personId"`
	Slot     string `json:"slot"`
}

func (r *crewShiftReq) parseDate() (time.Time, error) {
	return time.Parse("2006-01-02", r.WorkDate)
}

// shiftConflict 判断同工地同人同日是否已有排班（含与全天班互相冲突）
func (h *Handler) shiftConflict(siteID, personID uint, workDate time.Time, excludeID uint) bool {
	var count int64
	q := h.DB.Model(&models.CrewShift{}).
		Where("site_id = ? AND person_id = ? AND work_date = ?", siteID, personID, workDate)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

func (h *Handler) ListCrewShifts(c *gin.Context) {
	var shifts []models.CrewShift
	q := h.DB.Preload("Site").Preload("Person").Order("work_date asc, id asc")
	if siteID := c.Query("siteId"); siteID != "" {
		q = q.Where("site_id = ?", siteID)
	}
	if personID := c.Query("personId"); personID != "" {
		q = q.Where("person_id = ?", personID)
	}
	if start := c.Query("startDate"); start != "" {
		q = q.Where("work_date >= ?", start)
	}
	if end := c.Query("endDate"); end != "" {
		q = q.Where("work_date <= ?", end)
	}
	if err := q.Find(&shifts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shifts)
}

func (h *Handler) GetCrewShift(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var shift models.CrewShift
	if err := h.DB.Preload("Site").Preload("Person").First(&shift, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排班不存在"})
		return
	}
	c.JSON(http.StatusOK, shift)
}

func (h *Handler) CreateCrewShift(c *gin.Context) {
	var req crewShiftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if req.SiteID == 0 || req.PersonID == 0 || req.WorkDate == "" || !crewSlots[req.Slot] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工地、人员、日期与时段（morning/afternoon/full）必填"})
		return
	}
	workDate, err := req.parseDate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式应为 YYYY-MM-DD"})
		return
	}
	var site models.Site
	if err := h.DB.First(&site, req.SiteID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属工地不存在"})
		return
	}
	var person models.CrewPerson
	if err := h.DB.First(&person, req.PersonID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "排班人员不存在"})
		return
	}
	if h.shiftConflict(req.SiteID, req.PersonID, workDate, 0) {
		c.JSON(http.StatusConflict, gin.H{"error": "该人员在此工地当日已有排班"})
		return
	}
	shift := models.CrewShift{SiteID: req.SiteID, WorkDate: workDate, PersonID: req.PersonID, Slot: req.Slot}
	if err := h.DB.Create(&shift).Error; err != nil {
		if isDuplicateErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "该人员在此工地当日已有排班"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Site").Preload("Person").First(&shift, shift.ID)
	c.JSON(http.StatusCreated, shift)
}

func (h *Handler) UpdateCrewShift(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var shift models.CrewShift
	if err := h.DB.First(&shift, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排班不存在"})
		return
	}
	var req crewShiftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}
	if req.SiteID == 0 || req.PersonID == 0 || req.WorkDate == "" || !crewSlots[req.Slot] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工地、人员、日期与时段（morning/afternoon/full）必填"})
		return
	}
	workDate, err := req.parseDate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式应为 YYYY-MM-DD"})
		return
	}
	var site models.Site
	if err := h.DB.First(&site, req.SiteID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属工地不存在"})
		return
	}
	var person models.CrewPerson
	if err := h.DB.First(&person, req.PersonID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "排班人员不存在"})
		return
	}
	if h.shiftConflict(req.SiteID, req.PersonID, workDate, shift.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": "该人员在此工地当日已有排班"})
		return
	}
	shift.SiteID = req.SiteID
	shift.WorkDate = workDate
	shift.PersonID = req.PersonID
	shift.Slot = req.Slot
	if err := h.DB.Save(&shift).Error; err != nil {
		if isDuplicateErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "该人员在此工地当日已有排班"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Preload("Site").Preload("Person").First(&shift, shift.ID)
	c.JSON(http.StatusOK, shift)
}

func (h *Handler) DeleteCrewShift(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 物理删除：避免软删残留行占据 (site,date,person) 唯一索引
	if err := h.DB.Unscoped().Delete(&models.CrewShift{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// ---------- Overview ----------

func (h *Handler) Overview(c *gin.Context) {
	var siteCount, unitCount, findCount int64
	h.DB.Model(&models.Site{}).Count(&siteCount)
	h.DB.Model(&models.Unit{}).Count(&unitCount)
	h.DB.Model(&models.Find{}).Count(&findCount)

	type typeStat struct {
		ArtifactType string `json:"artifactType"`
		Count        int64  `json:"count"`
	}
	var byType []typeStat
	h.DB.Model(&models.Find{}).
		Select("artifact_type as artifact_type, count(*) as count").
		Group("artifact_type").
		Scan(&byType)

	c.JSON(http.StatusOK, gin.H{
		"siteCount": siteCount,
		"unitCount": unitCount,
		"findCount": findCount,
		"byType":    byType,
	})
}
