INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES
    (
        'fcau-application-form',
        'FCAU Application Form',
        'Form for applicants to submit their application for the FCAU process.',
        '{
            "type": "object",
            "required": [
                "field1"
            ],
            "properties": {
                "field1": {
                    "type": "string",
                    "title": "Field 1",
                    "example": "Example value for field 1"
                },
                "supportingDocuments": {
                    "type": "string",
                    "title": "Supporting Documents",
                    "format": "file"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Application Form",
                    "type": "Label"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/field1"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/supportingDocuments"
                }
            ]
        }',
        '1.0',
        TRUE
    ),
    (
        'fcau-lab-payment-form',
        'FCAU Lab Payment Form',
        'Form for applicants to upload proof of payment for lab testing.',
        '{
            "type": "object",
            "required": [
                "supportingDocuments"
            ],
            "properties": {
                "supportingDocuments": {
                    "type": "string",
                    "title": "Upload Proof of Payment",
                    "format": "file"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Upload Proof of Payment",
                    "type": "Label"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/supportingDocuments"
                }
            ]
        }',
        '1.0',
        TRUE
    ),
    (
        'fcau-application-review-response',
        'FCAU Application Review Response Form',
        'Form for reviewers to provide their decision and comments on the application.',
        '{
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["APPROVED", "REJECTED"]
                },
                "reviewer_comments": {
                    "type": "string"
                }
            },
            "required": ["decision"]
        }'::jsonb,
        '{}'::jsonb,
        1,
        true
    ),
    (
        'fcau-lab-payment-review-response',
        'FCAU Lab Payment Review Response Form',
        'Form for reviewers to provide their decision and comments on the lab payment.',
        '{
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["APPROVED", "REJECTED"]
                },
                "reviewer_comments": {
                    "type": "string"
                }
            },
            "required": ["decision"]
         }'::jsonb,
        '{}'::jsonb,
        1,
        true
    );