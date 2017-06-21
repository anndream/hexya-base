// Copyright 2016 NDP Systèmes. All Rights Reserved.
// See LICENSE file for full licensing details.

package base

import (
	"fmt"

	"github.com/hexya-erp/hexya/hexya/actions"
	"github.com/hexya-erp/hexya/hexya/models"
	"github.com/hexya-erp/hexya/hexya/models/security"
	"github.com/hexya-erp/hexya/hexya/models/types"
	"github.com/hexya-erp/hexya/pool"
)

// BaseAuthBackend is the authentication backend of the Base module
// Users are authenticated against the User model in the database
type BaseAuthBackend struct{}

// Authenticate the user defined by login and secret.
func (bab *BaseAuthBackend) Authenticate(login, secret string, context *types.Context) (uid int64, err error) {
	models.ExecuteInNewEnvironment(security.SuperUserID, func(env models.Environment) {
		uid, err = pool.User().NewSet(env).WithNewContext(context).Authenticate(login, secret)
	})
	return
}

func init() {
	cpWizard := pool.UserChangePasswordWizard().DeclareTransientModel()
	cpWizard.AddOne2ManyField("Users", models.ReverseFieldParams{RelationModel: pool.UserChangePasswordWizardLine(),
		ReverseFK: "Wizard", Default: func(env models.Environment, fMap models.FieldMap) interface{} {
			activeIds := env.Context().GetIntegerSlice("active_ids")
			userLines := pool.UserChangePasswordWizardLine().NewSet(env)
			for _, user := range pool.User().Search(env, pool.User().ID().In(activeIds)).Records() {
				ul := pool.UserChangePasswordWizardLine().Create(env, pool.UserChangePasswordWizardLineData{
					User:        user,
					UserLogin:   user.Login(),
					NewPassword: user.Password(),
				})
				userLines = userLines.Union(ul)
			}
			return userLines
		}})

	cpWizard.Methods().ChangePasswordButton().DeclareMethod(
		`ChangePasswordButton is called when the user clicks on 'Apply' button in the popup.
		It updates the user's password.`,
		func(rs pool.UserChangePasswordWizardSet) {
			for _, userLine := range rs.Users().Records() {
				userLine.User().SetPassword(userLine.NewPassword())
			}
		})

	cpWizardLine := pool.UserChangePasswordWizardLine().DeclareTransientModel()
	cpWizardLine.AddMany2OneField("Wizard", models.ForeignKeyFieldParams{RelationModel: pool.UserChangePasswordWizard()})
	cpWizardLine.AddMany2OneField("User", models.ForeignKeyFieldParams{RelationModel: pool.User(), OnDelete: models.Cascade})
	cpWizardLine.AddCharField("UserLogin", models.StringFieldParams{})
	cpWizardLine.AddCharField("NewPassword", models.StringFieldParams{})

	user := pool.User().DeclareModel()
	user.AddDateTimeField("LoginDate", models.SimpleFieldParams{})
	user.AddMany2OneField("Partner", models.ForeignKeyFieldParams{RelationModel: pool.Partner(), Embed: true})
	user.AddCharField("Login", models.StringFieldParams{Required: true})
	user.AddCharField("Password", models.StringFieldParams{})
	user.AddCharField("NewPassword", models.StringFieldParams{})
	user.AddTextField("Signature", models.StringFieldParams{})
	user.AddBooleanField("Active", models.SimpleFieldParams{})
	user.AddCharField("ActionID", models.StringFieldParams{GoType: new(actions.ActionRef)})
	user.AddMany2OneField("Company", models.ForeignKeyFieldParams{RelationModel: pool.Company()})
	user.AddMany2ManyField("Companies", models.Many2ManyFieldParams{RelationModel: pool.Company(), JSON: "company_ids"})
	user.AddBinaryField("ImageSmall", models.SimpleFieldParams{})
	user.AddMany2ManyField("Groups", models.Many2ManyFieldParams{RelationModel: pool.Group(), JSON: "group_ids"})

	user.Methods().Write().Extend("",
		func(rs pool.UserSet, data models.FieldMapper, fieldsToUnset ...models.FieldNamer) bool {
			res := rs.Super().Write(data, fieldsToUnset...)
			fMap := data.FieldMap(fieldsToUnset...)
			_, ok1 := fMap["Groups"]
			_, ok2 := fMap["group_ids"]
			if ok1 || ok2 {
				log.Debug("Updating user groups", "user", rs.Name(), "uid", rs.ID(), "groups", rs.Groups())
				// We get groups before removing all memberships otherwise we might get stuck with permissions if we
				// are modifying our own user memberships.
				groups := rs.Groups().Records()
				security.Registry.RemoveAllMembershipsForUser(rs.ID())
				for _, group := range groups {
					security.Registry.AddMembership(rs.ID(), security.Registry.GetGroup(group.GroupID()))
				}
			}
			return res
		})

	user.Methods().NameGet().Extend("",
		func(rs pool.UserSet) string {
			res := rs.Super().NameGet()
			return fmt.Sprintf("%s (%s)", res, rs.Login())
		})

	user.Methods().ContextGet().DeclareMethod(
		`UsersContextGet returns a context with the user's lang, tz and uid
		This method must be called on a singleton.`,
		func(rs pool.UserSet) *types.Context {
			rs.EnsureOne()
			res := types.NewContext()
			res = res.WithKey("lang", rs.Lang())
			res = res.WithKey("tz", rs.TZ())
			res = res.WithKey("uid", rs.ID())
			return res
		})

	user.Methods().HasGroup().DeclareMethod(
		`HasGroup returns true if this user belongs to the group with the given ID`,
		func(rs pool.UserSet, groupID string) bool {
			group := security.Registry.GetGroup(groupID)
			return security.Registry.HasMembership(rs.ID(), group)
		})

	user.Methods().Authenticate().DeclareMethod(
		"Authenticate the user defined by login and secret",
		func(rs pool.UserSet, login, secret string) (uid int64, err error) {
			user := rs.Search(pool.User().Login().Equals(login))
			if user.Len() == 0 {
				err = security.UserNotFoundError(login)
				return
			}
			if user.Password() != secret {
				err = security.InvalidCredentialsError(login)
				return
			}
			uid = user.ID()
			return
		})

	security.AuthenticationRegistry.RegisterBackend(new(BaseAuthBackend))

}