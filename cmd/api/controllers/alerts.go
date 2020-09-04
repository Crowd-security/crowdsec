package controllers

import (
	"github.com/crowdsecurity/crowdsec/cmd/api/ent/alert"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type CreateAlertInput struct {
	MachineId  int        `json:"machineId" binding:"required"`
	Scenario   string     `json:"scenario" binding:"required"`
	BucketId   string     `json:"bucketId" binding:"required"`
	Message    string     `json:"message" binding:"required"`
	EventCount int        `json:"eventCount" binding:"required"`
	StartedAt  time.Time  `json:"startedAt" binding:"required"`
	StoppedAt  time.Time  `json:"stoppedAt" binding:"required"`
	Capacity   int        `json:"capacity" binding:"required"`
	LeakSpeed  int        `json:"leakSpeed" binding:"required"`
	Reprocess  bool       `json:"reprocess"`
	Source     Source     `json:"source" binding:"required"`
	Events     []Event    `json:"events" binding:"required"`
	Metas      []Meta     `json:"metas"`
	Decisions  []Decision `json:"decisions" binding:"required"`
}

type Event struct {
	Time       time.Time `json:"time"`
	Serialized string    `json:"serialized"`
}

type Source struct {
	Scope     string  `json:"scope" binding:"required"`
	Value     string  `json:"value" binding:"required"`
	Ip        string  `json:"ip"`
	Range     string  `json:"range"`
	AsNumber  string  `json:"as_number"`
	AsName    string  `json:"as_name"`
	Country   string  `json:"country"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

type Meta struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Decision struct {
	Until         time.Time `json:"until"`
	Scenario      string    `json:"scenario"`
	DecisionType  string    `json:"decisionType"`
	SourceIpStart int       `json:"sourceIpStart"`
	SourceIpEnd   int       `json:"sourceIpEnd"`
	SourceValue   string    `json:"sourceValue"`
	SourceScope   string    `json:"sourceScope"`
}

func (c *Controller) CreateAlert(gctx *gin.Context) {
	var input CreateAlertInput
	if err := gctx.ShouldBindJSON(&input); err != nil {
		gctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	machine, err := QueryMachine(c.Ectx, c.Client, input.MachineId)
	if err != nil {
		log.Errorf("failed query machine: %v", err)
		gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating alert, machineId not exist"})
		return
	}

	alert, err := c.Client.Alert.
		Create().
		SetScenario(input.Scenario).
		SetBucketId(input.BucketId).
		SetMessage(input.Message).
		SetEventsCount(input.EventCount).
		SetStartedAt(input.StartedAt).
		SetStoppedAt(input.StoppedAt).
		SetSourceScope(input.Source.Scope).
		SetSourceValue(input.Source.Value).
		SetSourceIp(input.Source.Ip).
		SetSourceRange(input.Source.Range).
		SetSourceAsNumber(input.Source.AsNumber).
		SetSourceAsName(input.Source.AsName).
		SetSourceCountry(input.Source.Country).
		SetSourceLatitude(input.Source.Latitude).
		SetSourceLongitude(input.Source.Longitude).
		SetCapacity(input.Capacity).
		SetLeakSpeed(input.LeakSpeed).
		SetReprocess(input.Reprocess).
		SetOwner(machine).
		Save(c.Ectx)
	if err != nil {
		log.Errorf("failed creating alert: %v", err)
		gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating alert"})
		return
	}

	if len(input.Events) > 0 {
		for _, eventItem := range input.Events {
			_, err := c.Client.Event.
				Create().
				SetTime(eventItem.Time).
				SetSerialized(eventItem.Serialized).
				SetOwner(alert).
				Save(c.Ectx)
			if err != nil {
				log.Errorf("failed creating event: %v", err)
				gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating alert"})
				return
			}
		}
	}

	if len(input.Metas) > 0 {
		for _, metaItem := range input.Metas {
			_, err := c.Client.Meta.
				Create().
				SetKey(metaItem.Key).
				SetValue(metaItem.Value).
				SetOwner(alert).
				Save(c.Ectx)
			if err != nil {
				log.Errorf("failed creating meta: %v", err)
				gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating alert"})
				return
			}
		}
	}

	if len(input.Decisions) > 0 {
		for _, decisionItem := range input.Decisions {
			_, err := c.Client.Decision.
				Create().
				SetUntil(decisionItem.Until).
				SetScenario(decisionItem.Scenario).
				SetDecisionType(decisionItem.DecisionType).
				SetSourceIpStart(decisionItem.SourceIpStart).
				SetSourceIpEnd(decisionItem.SourceIpEnd).
				SetSourceValue(decisionItem.SourceValue).
				SetSourceScope(decisionItem.SourceScope).
				SetOwner(alert).
				Save(c.Ectx)
			if err != nil {
				log.Errorf("failed creating decision: %v", err)
				gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating alert"})
				return
			}
		}
	}

	gctx.JSON(http.StatusOK, gin.H{"data": alert})
	return
}

func (c *Controller) FindAlerts(gctx *gin.Context) {
	scenario := gctx.Query("scenario")
	sourceScope := gctx.Query("sourceScope")
	sourceValue := gctx.Query("sourceValue")

	alerts, err := c.Client.Debug().Alert.Query().
		Where(alert.And(
			alert.ScenarioContains(scenario),
			alert.SourceScopeContains(sourceScope),
			alert.SourceValueContains(sourceValue),
		)).
		WithDecisions().
		WithEvents().
		WithMetas().
		All(c.Ectx)
	if err != nil {
		log.Errorf("failed querying alert: %v", err)
		gctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed querying alert"})
		return
	}

	gctx.JSON(http.StatusOK, gin.H{"data": alerts})
	return
}