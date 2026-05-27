-- Project model policy 是 project 对模型可见性的覆盖策略。
CREATE TABLE project_model_policies (
    -- project_id: 策略所属项目 ID。--
    project_id BIGINT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,

    -- model_id: 策略作用的模型 ID。--
    model_id BIGINT NOT NULL REFERENCES models (id) ON DELETE CASCADE,

    -- visibility: 模型对该 project 的可见性。--
    visibility TEXT NOT NULL CHECK (visibility IN ('allowed', 'denied')),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 同一 project 对同一模型只能有一条可见性策略。--
    PRIMARY KEY (project_id, model_id)
);

-- 查询 project 可见模型时会按 project 和 visibility 过滤。
CREATE INDEX idx_project_model_policies_project_visibility
    ON project_model_policies (project_id, visibility);

-- 模型下线或排查策略时需要按 model_id 反查引用。
CREATE INDEX idx_project_model_policies_model_id
    ON project_model_policies (model_id);
