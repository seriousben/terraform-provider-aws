package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pkg/errors"
)

func resourceAwsCognitoUserPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCognitoUserPoolCreate,
		Read:   resourceAwsCognitoUserPoolRead,
		Update: resourceAwsCognitoUserPoolUpdate,
		Delete: resourceAwsCognitoUserPoolDelete,

		Schema: map[string]*schema.Schema{
			"alias_attributes": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCognitoUserPoolAliasAttribute,
				},
			},

			"auto_verified_attributes": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCognitoUserPoolAutoVerifiedAttribute,
				},
			},

			"email_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 0,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"reply_to_email_address": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateCognitoUserPoolReplyEmailAddress,
						},
						"source_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"email_verification_subject": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolEmailVerificationSubject,
			},

			"email_verification_message": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolEmailVerificationMessage,
			},

			"mfa_configuration": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      cognitoidentityprovider.UserPoolMfaTypeOff,
				ValidateFunc: validateCognitoUserPoolMfaConfiguration,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sms_authentication_message": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolSmsAuthenticationMessage,
			},

			"sms_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 0,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"external_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"sns_caller_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"sms_verification_message": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolSmsVerificationMessage,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsCognitoUserPoolCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("alias_attributes"); ok {
		params.AliasAttributes = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("auto_verified_attributes"); ok {
		params.AutoVerifiedAttributes = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("email_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if !ok {
			return errors.New("email_configuration is <nil>")
		}

		if config != nil {
			emailConfigurationType := &cognitoidentityprovider.EmailConfigurationType{}

			if v, ok := config["reply_to_email_address"]; ok && v.(string) != "" {
				emailConfigurationType.ReplyToEmailAddress = aws.String(v.(string))
			}

			if v, ok := config["source_arn"]; ok && v.(string) != "" {
				emailConfigurationType.SourceArn = aws.String(v.(string))
			}

			params.EmailConfiguration = emailConfigurationType
		}
	}

	if v, ok := d.GetOk("email_verification_subject"); ok {
		params.EmailVerificationSubject = aws.String(v.(string))
	}

	if v, ok := d.GetOk("email_verification_message"); ok {
		params.EmailVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("mfa_configuration"); ok {
		params.MfaConfiguration = aws.String(v.(string))
	}

	if v, ok := d.GetOk("sms_authentication_message"); ok {
		params.SmsAuthenticationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("sms_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if !ok {
			return errors.New("sms_configuration is <nil>")
		}

		if config != nil {
			smsConfigurationType := &cognitoidentityprovider.SmsConfigurationType{
				SnsCallerArn: aws.String(config["sns_caller_arn"].(string)),
			}

			if v, ok := config["external_id"]; ok && v.(string) != "" {
				smsConfigurationType.ExternalId = aws.String(v.(string))
			}

			params.SmsConfiguration = smsConfigurationType
		}
	}

	if v, ok := d.GetOk("sms_verification_message"); ok {
		params.SmsVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.UserPoolTags = tagsFromMapGeneric(v.(map[string]interface{}))
	}
	log.Printf("[DEBUG] Creating Cognito User Pool: %s", params)

	resp, err := conn.CreateUserPool(params)

	if err != nil {
		return errwrap.Wrapf("Error creating Cognito User Pool: {{err}}", err)
	}

	d.SetId(*resp.UserPool.Id)

	return resourceAwsCognitoUserPoolRead(d, meta)
}

func resourceAwsCognitoUserPoolRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Cognito User Pool: %s", params)

	resp, err := conn.DescribeUserPool(params)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] Cognito User Pool %s is already gone", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if resp.UserPool.AliasAttributes != nil {
		d.Set("alias_attributes", flattenStringList(resp.UserPool.AliasAttributes))
	}
	if resp.UserPool.AutoVerifiedAttributes != nil {
		d.Set("auto_verified_attributes", flattenStringList(resp.UserPool.AutoVerifiedAttributes))
	}
	if resp.UserPool.EmailVerificationSubject != nil {
		d.Set("email_verification_subject", *resp.UserPool.EmailVerificationSubject)
	}
	if resp.UserPool.EmailVerificationMessage != nil {
		d.Set("email_verification_message", *resp.UserPool.EmailVerificationMessage)
	}
	if resp.UserPool.MfaConfiguration != nil {
		d.Set("mfa_configuration", *resp.UserPool.MfaConfiguration)
	}
	if resp.UserPool.SmsVerificationMessage != nil {
		d.Set("sms_verification_message", *resp.UserPool.SmsVerificationMessage)
	}
	if resp.UserPool.SmsAuthenticationMessage != nil {
		d.Set("sms_authentication_message", *resp.UserPool.SmsAuthenticationMessage)
	}

	if err := d.Set("email_configuration", flattenCognitoUserPoolEmailConfiguration(resp.UserPool.EmailConfiguration)); err != nil {
		return errwrap.Wrapf("Failed setting email_configuration: {{err}}", err)
	}

	if err := d.Set("sms_configuration", flattenCognitoUserPoolSmsConfiguration(resp.UserPool.SmsConfiguration)); err != nil {
		return errwrap.Wrapf("Failed setting sms_configuration: {{err}}", err)
	}

	d.Set("tags", tagsToMapGeneric(resp.UserPool.UserPoolTags))

	return nil
}

func resourceAwsCognitoUserPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.UpdateUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	// TODO - Handle update of AliasAttributes

	if d.HasChange("auto_verified_attributes") {
		params.AutoVerifiedAttributes = expandStringList(d.Get("auto_verified_attributes").([]interface{}))
	}

	if d.HasChange("email_configuration") {
		configs := d.Get("email_configuration").([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if !ok {
			return errors.New("email_configuration is <nil>")
		}

		if config != nil {
			emailConfigurationType := &cognitoidentityprovider.EmailConfigurationType{}

			if v, ok := config["reply_to_email_address"]; ok && v.(string) != "" {
				emailConfigurationType.ReplyToEmailAddress = aws.String(v.(string))
			}

			if v, ok := config["source_arn"]; ok && v.(string) != "" {
				emailConfigurationType.SourceArn = aws.String(v.(string))
			}

			params.EmailConfiguration = emailConfigurationType
		}
	}

	if d.HasChange("email_verification_subject") {
		params.EmailVerificationSubject = aws.String(d.Get("email_verification_subject").(string))
	}

	if d.HasChange("email_verification_message") {
		params.EmailVerificationMessage = aws.String(d.Get("email_verification_message").(string))
	}

	if d.HasChange("mfa_configuration") {
		params.MfaConfiguration = aws.String(d.Get("mfa_configuration").(string))
	}

	if d.HasChange("sms_authentication_message") {
		params.SmsAuthenticationMessage = aws.String(d.Get("sms_authentication_message").(string))
	}

	if d.HasChange("sms_configuration") {
		configs := d.Get("sms_configuration").([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if !ok {
			return errors.New("sms_configuration is <nil>")
		}

		if config != nil {
			smsConfigurationType := &cognitoidentityprovider.SmsConfigurationType{
				SnsCallerArn: aws.String(config["sns_caller_arn"].(string)),
			}

			if v, ok := config["external_id"]; ok && v.(string) != "" {
				smsConfigurationType.ExternalId = aws.String(v.(string))
			}

			params.SmsConfiguration = smsConfigurationType
		}
	}

	if d.HasChange("sms_verification_message") {
		params.SmsVerificationMessage = aws.String(d.Get("sms_verification_message").(string))
	}

	if d.HasChange("tags") {
		params.UserPoolTags = tagsFromMapGeneric(d.Get("tags").(map[string]interface{}))
	}

	log.Printf("[DEBUG] Updating Cognito User Pool: %s", params)

	_, err := conn.UpdateUserPool(params)
	if err != nil {
		return errwrap.Wrapf("Error updating Cognito User pool: {{err}}", err)
	}

	return resourceAwsCognitoUserPoolRead(d, meta)
}

func resourceAwsCognitoUserPoolDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Cognito User Pool: %s", params)

	_, err := conn.DeleteUserPool(params)

	if err != nil {
		return errwrap.Wrapf("Error deleting user pool: {{err}}", err)
	}

	return nil
}