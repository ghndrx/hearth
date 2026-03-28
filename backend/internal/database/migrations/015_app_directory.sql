-- +migrate Up

-- Apps table (App Directory / Bot Marketplace)
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200) NOT NULL,
    long_description TEXT,
    developer_id UUID NOT NULL REFERENCES users(id),
    oauth_app_id UUID REFERENCES oauth_apps(id),
    category INT NOT NULL,
    tags TEXT[],
    icon_url TEXT,
    screenshots TEXT[],
    install_count INT DEFAULT 0,
    rating DECIMAL(2,1) DEFAULT 0.0,
    review_count INT DEFAULT 0,
    status INT DEFAULT 0,
    privacy_policy_url TEXT,
    terms_of_service_url TEXT,
    support_server_id UUID REFERENCES servers(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_apps_developer ON apps(developer_id);
CREATE INDEX idx_apps_category ON apps(category);
CREATE INDEX idx_apps_status ON apps(status);
CREATE INDEX idx_apps_rating ON apps(rating DESC);
CREATE INDEX idx_apps_install_count ON apps(install_count DESC);

-- App installations (which servers have installed which apps)
CREATE TABLE app_installations (
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    installer_id UUID NOT NULL REFERENCES users(id),
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (app_id, server_id)
);

CREATE INDEX idx_app_installations_server ON app_installations(server_id);
CREATE INDEX idx_app_installations_installer ON app_installations(installer_id);

-- App reviews
CREATE TABLE app_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_id, user_id)
);

CREATE INDEX idx_app_reviews_app ON app_reviews(app_id);
CREATE INDEX idx_app_reviews_user ON app_reviews(user_id);
CREATE INDEX idx_app_reviews_rating ON app_reviews(rating DESC);

-- App developer teams (who has access to manage an app)
CREATE TABLE app_developer_teams (
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    PRIMARY KEY (app_id, user_id)
);

CREATE INDEX idx_app_developer_teams_user ON app_developer_teams(user_id);

-- +migrate Down

DROP TABLE IF EXISTS app_developer_teams;
DROP TABLE IF EXISTS app_reviews;
DROP TABLE IF EXISTS app_installations;
DROP TABLE IF EXISTS apps;
