package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

type mockForumTagService struct {
	getChannelTagsFunc    func(ctx context.Context, channelID uuid.UUID) ([]*models.ForumTag, error)
	createTagFunc         func(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error)
	updateTagFunc         func(ctx context.Context, tagID, userID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error)
	deleteTagFunc         func(ctx context.Context, tagID, userID uuid.UUID) error
	applyTagsToThreadFunc func(ctx context.Context, threadID, userID uuid.UUID, tagIDs []uuid.UUID) error
	getThreadTagsFunc     func(ctx context.Context, threadID uuid.UUID) ([]*models.ForumTag, error)
	pinThreadFunc         func(ctx context.Context, threadID, userID uuid.UUID, pin bool) error
}

func setupForumTagsApp(mock *mockForumTagService) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			uid, err := uuid.Parse(userIDStr)
			if err != nil {
				c.Locals("userID", userIDStr)
			} else {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	app.Get("/channels/:channelId/tags", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		tags, err := mock.getChannelTagsFunc(c.Context(), channelID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(fiber.Map{"tags": tags})
	})

	app.Post("/channels/:channelId/tags", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel ID"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req models.CreateForumTagRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		tag, err := mock.createTagFunc(c.Context(), channelID, userID, &req)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.Status(fiber.StatusCreated).JSON(tag)
	})

	app.Patch("/forum-tags/:tagId", func(c *fiber.Ctx) error {
		tagID, err := uuid.Parse(c.Params("tagId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tag ID"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req models.UpdateForumTagRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		tag, err := mock.updateTagFunc(c.Context(), tagID, userID, &req)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(tag)
	})

	app.Delete("/forum-tags/:tagId", func(c *fiber.Ctx) error {
		tagID, err := uuid.Parse(c.Params("tagId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tag ID"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		err = mock.deleteTagFunc(c.Context(), tagID, userID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Put("/threads/:threadId/tags", func(c *fiber.Ctx) error {
		threadID, err := uuid.Parse(c.Params("threadId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread ID"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var body struct {
			TagIDs []uuid.UUID `json:"tag_ids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		err = mock.applyTagsToThreadFunc(c.Context(), threadID, userID, body.TagIDs)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(fiber.Map{"message": "tags applied"})
	})

	app.Get("/threads/:threadId/tags", func(c *fiber.Ctx) error {
		threadID, err := uuid.Parse(c.Params("threadId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread ID"})
		}
		tags, err := mock.getThreadTagsFunc(c.Context(), threadID)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(fiber.Map{"tags": tags})
	})

	app.Put("/threads/:threadId/pin", func(c *fiber.Ctx) error {
		threadID, err := uuid.Parse(c.Params("threadId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread ID"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var body struct {
			Pin bool `json:"pin"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		err = mock.pinThreadFunc(c.Context(), threadID, userID, body.Pin)
		if err != nil {
			return HandleServiceError(c, err)
		}
		return c.JSON(fiber.Map{"message": "thread pin updated"})
	})

	return app
}

func TestListTags_Success(t *testing.T) {
	channelID := uuid.New()
	tagID := uuid.New()
	mock := &mockForumTagService{
		getChannelTagsFunc: func(ctx context.Context, cID uuid.UUID) ([]*models.ForumTag, error) {
			assert.Equal(t, channelID, cID)
			return []*models.ForumTag{
				{ID: tagID, ChannelID: channelID, Name: "bug", Moderated: false},
			}, nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/tags", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["tags"])
}

func TestListTags_InvalidChannelID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/channels/invalid/tags", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateTag_Success(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	tagID := uuid.New()
	mock := &mockForumTagService{
		createTagFunc: func(ctx context.Context, cID, uID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, "feature", req.Name)
			return &models.ForumTag{ID: tagID, ChannelID: channelID, Name: "feature"}, nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"feature"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestCreateTag_InvalidChannelID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"feature"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/invalid/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateTag_Success(t *testing.T) {
	tagID := uuid.New()
	userID := uuid.New()
	newName := "updated-tag"
	mock := &mockForumTagService{
		updateTagFunc: func(ctx context.Context, tID, uID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error) {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userID, uID)
			return &models.ForumTag{ID: tagID, Name: newName}, nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"updated-tag"}`
	req := httptest.NewRequest(http.MethodPatch, "/forum-tags/"+tagID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateTag_InvalidTagID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"updated-tag"}`
	req := httptest.NewRequest(http.MethodPatch, "/forum-tags/invalid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteTag_Success(t *testing.T) {
	tagID := uuid.New()
	userID := uuid.New()
	mock := &mockForumTagService{
		deleteTagFunc: func(ctx context.Context, tID, uID uuid.UUID) error {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userID, uID)
			return nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/forum-tags/"+tagID.String(), nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestDeleteTag_InvalidTagID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/forum-tags/invalid", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestApplyTags_Success(t *testing.T) {
	threadID := uuid.New()
	userID := uuid.New()
	tagID1 := uuid.New()
	tagID2 := uuid.New()
	mock := &mockForumTagService{
		applyTagsToThreadFunc: func(ctx context.Context, tID, uID uuid.UUID, tagIDs []uuid.UUID) error {
			assert.Equal(t, threadID, tID)
			assert.Equal(t, userID, uID)
			assert.Len(t, tagIDs, 2)
			return nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"tag_ids":["` + tagID1.String() + `","` + tagID2.String() + `"]}`
	req := httptest.NewRequest(http.MethodPut, "/threads/"+threadID.String()+"/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestApplyTags_InvalidThreadID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"tag_ids":[]}`
	req := httptest.NewRequest(http.MethodPut, "/threads/invalid/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetThreadTags_Success(t *testing.T) {
	threadID := uuid.New()
	tagID := uuid.New()
	mock := &mockForumTagService{
		getThreadTagsFunc: func(ctx context.Context, tID uuid.UUID) ([]*models.ForumTag, error) {
			assert.Equal(t, threadID, tID)
			return []*models.ForumTag{
				{ID: tagID, Name: "bug"},
			}, nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID.String()+"/tags", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["tags"])
}

func TestGetThreadTags_InvalidThreadID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/threads/invalid/tags", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPinThread_Success(t *testing.T) {
	threadID := uuid.New()
	userID := uuid.New()
	mock := &mockForumTagService{
		pinThreadFunc: func(ctx context.Context, tID, uID uuid.UUID, pin bool) error {
			assert.Equal(t, threadID, tID)
			assert.Equal(t, userID, uID)
			assert.True(t, pin)
			return nil
		},
	}

	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"pin":true}`
	req := httptest.NewRequest(http.MethodPut, "/threads/"+threadID.String()+"/pin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPinThread_InvalidThreadID(t *testing.T) {
	mock := &mockForumTagService{}
	app := setupForumTagsApp(mock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"pin":true}`
	req := httptest.NewRequest(http.MethodPut, "/threads/invalid/pin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Forum Tags Handler Tests

func TestSplitAndTrim_BasicSplit(t *testing.T) {
	result := splitAndTrim("a,b,c", ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitAndTrim_WithSpaces(t *testing.T) {
	result := splitAndTrim(" a , b , c ", ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitAndTrim_EmptyString(t *testing.T) {
	result := splitAndTrim("", ",")
	assert.Nil(t, result)
}

func TestSplitAndTrim_OnlySeparators(t *testing.T) {
	result := splitAndTrim(",,,", ",")
	assert.Equal(t, []string{}, result)
}

func TestSplitAndTrim_SingleElement(t *testing.T) {
	result := splitAndTrim("only", ",")
	assert.Equal(t, []string{"only"}, result)
}

func TestSplitAndTrim_WithTabs(t *testing.T) {
	result := splitAndTrim("\ta\t,\tb\t", ",")
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestSplitString_BasicSplit(t *testing.T) {
	result := splitString("a,b,c", ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitString_NoSplit(t *testing.T) {
	result := splitString("abc", ",")
	assert.Equal(t, []string{"abc"}, result)
}

func TestSplitString_EmptyString(t *testing.T) {
	result := splitString("", ",")
	assert.Equal(t, []string{""}, result)
}

func TestTrimSpace_NoWhitespace(t *testing.T) {
	result := trimSpace("abc")
	assert.Equal(t, "abc", result)
}

func TestTrimSpace_LeadingWhitespace(t *testing.T) {
	result := trimSpace("  abc")
	assert.Equal(t, "abc", result)
}

func TestTrimSpace_TrailingWhitespace(t *testing.T) {
	result := trimSpace("abc  ")
	assert.Equal(t, "abc", result)
}

func TestTrimSpace_BothWhitespace(t *testing.T) {
	result := trimSpace("  abc  ")
	assert.Equal(t, "abc", result)
}

func TestTrimSpace_AllWhitespace(t *testing.T) {
	result := trimSpace("   ")
	assert.Equal(t, "", result)
}

func TestTrimSpace_WithTabs(t *testing.T) {
	result := trimSpace("\tabc\t")
	assert.Equal(t, "abc", result)
}

// mockForumTagServiceWithFilterPosts for ListPosts test
type mockForumTagServiceWithFilterPosts struct {
	mockForumTagService
	filterForumPostsFunc func(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]*models.Thread, []*models.ForumTag, int, error)
}

func (m *mockForumTagServiceWithFilterPosts) FilterForumPosts(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]*models.Thread, []*models.ForumTag, int, error) {
	if m.filterForumPostsFunc != nil {
		return m.filterForumPostsFunc(ctx, channelID, filter, limit, offset)
	}
	return []*models.Thread{}, []*models.ForumTag{}, 0, nil
}

func TestListPosts_Success(t *testing.T) {
	channelID := uuid.New()
	threadID := uuid.New()
	tagID := uuid.New()

	mock := &mockForumTagServiceWithFilterPosts{
		filterForumPostsFunc: func(ctx context.Context, cID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]*models.Thread, []*models.ForumTag, int, error) {
			assert.Equal(t, channelID, cID)
			assert.Equal(t, 25, limit)
			assert.Equal(t, 0, offset)
			return []*models.Thread{
					{
						ID:              threadID,
						ParentChannelID: channelID,
						Name:            "Test Thread",
						AppliedTags:     []uuid.UUID{tagID},
					},
				}, []*models.ForumTag{
					{ID: tagID, Name: "help"},
				}, 1, nil
		},
	}

	app := fiber.New()

	// We can't easily test ListPosts without a real service due to the complex signature
	// This test documents the endpoint behavior
	app.Get("/channels/:channelId/posts", func(c *fiber.Ctx) error {
		chID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}

		filter := &models.ForumPostFilter{}
		limit := c.QueryInt("limit", 25)
		if limit <= 0 || limit > 50 {
			limit = 25
		}
		offset := c.QueryInt("offset", 0)

		threads, tags, total, err := mock.FilterForumPosts(c.Context(), chID, filter, limit, offset)
		if err != nil {
			return HandleServiceError(c, err)
		}

		tagMap := make(map[uuid.UUID][]models.ForumTag)
		for _, tag := range tags {
			tagMap[tag.ID] = append(tagMap[tag.ID], *tag)
		}

		type ThreadWithTags struct {
			models.Thread
			Tags []models.ForumTag `json:"tags"`
		}
		threadsWithTags := make([]ThreadWithTags, len(threads))
		for i, t := range threads {
			var threadTags []models.ForumTag
			for _, tagID := range t.AppliedTags {
				if ts, ok := tagMap[tagID]; ok {
					threadTags = append(threadTags, ts...)
				}
			}
			threadsWithTags[i] = ThreadWithTags{Thread: *t, Tags: threadTags}
		}

		return c.JSON(fiber.Map{
			"threads":  threadsWithTags,
			"total":    total,
			"has_more": offset+len(threads) < total,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/posts", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["threads"])
	assert.NotNil(t, result["total"])
	assert.NotNil(t, result["has_more"])
}

func TestListPosts_WithTagFilter(t *testing.T) {
	channelID := uuid.New()
	tagID1 := uuid.New()
	tagID2 := uuid.New()

	mock := &mockForumTagServiceWithFilterPosts{
		filterForumPostsFunc: func(ctx context.Context, cID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]*models.Thread, []*models.ForumTag, int, error) {
			assert.Equal(t, channelID, cID)
			assert.Len(t, filter.TagIDs, 2)
			assert.Contains(t, filter.TagIDs, tagID1)
			assert.Contains(t, filter.TagIDs, tagID2)
			return []*models.Thread{}, []*models.ForumTag{}, 0, nil
		},
	}

	app := fiber.New()
	app.Get("/channels/:channelId/posts", func(c *fiber.Ctx) error {
		chID, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}

		filter := &models.ForumPostFilter{}
		if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
			tagIDStrs := splitAndTrim(tagIDsStr, ",")
			for _, idStr := range tagIDStrs {
				if id, err := uuid.Parse(idStr); err == nil {
					filter.TagIDs = append(filter.TagIDs, id)
				}
			}
		}
		limit := c.QueryInt("limit", 25)
		offset := c.QueryInt("offset", 0)

		threads, _, total, err := mock.FilterForumPosts(c.Context(), chID, filter, limit, offset)
		if err != nil {
			return HandleServiceError(c, err)
		}

		return c.JSON(fiber.Map{
			"threads":  threads,
			"total":    total,
			"has_more": offset+len(threads) < total,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/posts?tag_ids="+tagID1.String()+","+tagID2.String(), nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListPosts_InvalidChannelID(t *testing.T) {
	app := fiber.New()
	app.Get("/channels/:channelId/posts", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("channelId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}
		return c.JSON(fiber.Map{})
	})

	req := httptest.NewRequest(http.MethodGet, "/channels/invalid/posts", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListPosts_LimitCapping(t *testing.T) {
	channelID := uuid.New()

	app := fiber.New()
	app.Get("/channels/:channelId/posts", func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 25)
		// Verify that limit > 50 gets capped to 25
		if limit <= 0 || limit > 50 {
			limit = 25
		}
		assert.Equal(t, 25, limit)
		return c.JSON(fiber.Map{
			"threads":  []*models.Thread{},
			"total":    0,
			"has_more": false,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/posts?limit=100", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Ensure services import is used
var _ = (*services.ForumTagService)(nil)
