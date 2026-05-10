-- Migration: 019_taskv2_schema_align.down.sql
-- Description: Revert task_workflow_tasks to the pre-nsw-task-flow shape.

BEGIN;

DROP TABLE IF EXISTS task_workflow_tasks;

CREATE TABLE task_workflow_tasks (
    task_id text NOT NULL PRIMARY KEY,
    macro_workflow_id text NOT NULL,
    task_template_id text NOT NULL,
    state varchar(50) NOT NULL,
    data jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_macro_workflow_id ON task_workflow_tasks (macro_workflow_id);
CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_task_template_id ON task_workflow_tasks (task_template_id);
CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_state ON task_workflow_tasks (state);

COMMIT;