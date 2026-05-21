-- project_model_policies 表示 project 对某个模型的可见性覆盖策略。
-- 默认没有策略时继承全局 enabled 模型；一旦 project 存在 allowed 策略，则进入 allow-list 模式。
CREATE TABLE project_model_policies (
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    model_id BIGINT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    visibility TEXT NOT NULL CHECK ( visibility IN ('allowed', 'denied')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, model_id)
);

CREATE INDEX idx_project_model_policies_project_visibility
    ON project_model_policies(project_id, visibility);

CREATE INDEX idx_project_model_policies_model_id
    ON project_model_policies(model_id);