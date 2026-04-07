INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES
    -- FCAU Application Submission Task
    (
        'fcau:application_submission',
        'Application Submission',
        'Task for applicants to submit their application for the FCAU process',
        'SIMPLE_FORM',
        ('{
            "agency": "FCAU",
            "formId": "fcau-application-form",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "fcau-application-review-response"
                    },
                    "mapping": {
                        "decision": "fcau:application_decision",
                        "reviewer_comments": "fcau:reviewer_comments"
                    }
                },
                "transition": {
                    "field": "decision",
                    "default": "OGA_VERIFICATION_REJECTED",
                    "mapping": {
                        "APPROVED": "OGA_VERIFICATION_APPROVED"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "SIMPLE_FORM",
                        "templateKey": "fcau:general_application:v1"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Sample Drop Task
    (
        'fcau:sample_drop',
        'Sample Drop-off',
        'Task for applicants to confirm they have dropped off their sample at the designated location.',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Awaiting Sample Drop-off Confirmation",
                "description": "If you still have not dropped off your sample yet, please do so at your earliest convenience. Once you have dropped this work, we will notify you"
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "npqs:sample_drop_ack:v1"
                    },
                    "template": {
                        "consignee_name": "abcde",
                        "consigneeAddress": "123 Sample Street, Sample City, Country"
                    }
                },
                "response": {
                    "display": {
                        "formId": "npqs-sample-drop-ack-response"
                    },
                    "mapping": {
                        "decision": "sample_drop_confirmed"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Testing Requirement Task
    (
        'fcau:testing_requirement',
        'Testing Requirement Assessment',
        'Task to determine if the applicant requires lab testing based on their application details.',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Testing Requirement",
                "description": ""
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "npqs:testing_requirement:v1"
                    },
                    "template": {
                        "consignee_name": "abcde",
                        "consigneeAddress": "123 Sample Street, Sample City, Country"
                    }
                },
                "response": {
                    "display": {
                        "formId": "testing-requirement-response"
                    },
                    "mapping": {
                        "decision": "requires_lab_test"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Lab Payment Upload Task
    (
        'fcau:lab_payment_upload',
        'Lab Payment Upload',
        'Task for applicants to upload proof of payment for lab testing.',
        'SIMPLE_FORM',
        ('{
            "agency": "FCAU",
            "formId": "fcau-lab-payment-form",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "fcau-lab-payment-review-response"
                    },
                    "mapping": {
                        "decision": "fcau:lab_payment_decision",
                        "reviewer_comments": "fcau:reviewer_comments"
                    }
                },
                "transition": {
                    "field": "decision",
                    "default": "OGA_VERIFICATION_REJECTED",
                    "mapping": {
                        "APPROVED": "OGA_VERIFICATION_APPROVED"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "SIMPLE_FORM",
                        "templateKey": "fcau:lab_payment_upload:v1"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Lab Results Review Task
    (
        'fcau:lab_results_review',
        'Lab Results Review',
        'Task for lab personnel to review the test results and make a decision.',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Lab Results Review",
                "description": ""
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "npqs:lab_results_review:v1"
                    },
                    "template": {
                        "consignee_name": "abcde",
                        "consigneeAddress": "123 Sample Street, Sample City, Country"
                    }
                },
                "response": {
                    "display": {
                        "formId": "lab-results-review-response"
                    },
                    "mapping": {
                        "decision": "lab_decision"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Payment Task
    (
        'fcau:payment',
        'Payment',
        'Task for applicants to make payment for the FCAU process.',
        'PAYMENT',
        '{
            "currency": "LKR",
            "ttl": 3600,
            "orgId": "CUSTOMS",
            "serviceType": "CUSTOMS DECLARATION",
            "breakdown": [
            {
                "description": "Levy Payment for {cusdec.id}",
                "category": "ADDITION",
                "type": "FIXED",
                "quantity": "{gx_quantity_levy:1}",
                "unitPrice": "{cusdec.cess:345}"
            },
            {
                "description": "Processing Fee",
                "category": "ADDITION",
                "type": "FIXED",
                "quantity": "1",
                "unitPrice": "500.00"
            },
            {
                "description": "Exemption",
                "category": "DEDUCTION",
                "type": "PERCENTAGE",
                "value": "5"
            },
            {
                "description": "VAT",
                "category": "ADDITION",
                "type": "PERCENTAGE",
                "value": "{vat_rate:15}"
            }
            ]
        }',
        '[]'
    ),

    -- FCAU Certificate Issue Task
    (
        'fcau:certificate_issue',
        'Certificate Issuance',
        'Task for issuing the certificate to the applicant upon successful completion of the process.',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Certificate Issuance",
                "description": ""
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "npqs:manual_inspection:v1"
                    },
                    "template": {
                        "consignee_name": "abcde",
                        "consigneeAddress": "123 Sample Street, Sample City, Country"
                    }
                },
                "response": {
                    "display": {
                        "formId": "f1a00001-0001-4000-c000-000000000002"
                    },
                    "mapping": {
                        "inspectionDecision": "npqs:manual_inspection:decision",
                        "inspectorRemarks": "npqs:manual_inspection:remarks"
                    }
                }
            }
        }')::jsonb,
        '[]'
    );

