package manifest

type DeployTime struct {
	Release
}

func Decode(labels map[string]string, templateData any) (d DeployTime, err error) {
	s := Init()

	if d, err = s.decode(labels); err != nil {
		return
	}

	if err = d.template(templateData); err != nil {
		return
	}

	return
}

func (r *Release) decode(labels map[string]string) (d DeployTime, err error) {
	if err = r.Schema.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Name.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Branch.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Sha.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Origin.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Role.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Policy.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Resources.Decode(labels); err != nil {
		return d, err
	}

	if err = r.Bus.Decode(labels); err != nil {
		return d, err
	}

	return DeployTime{Release: *r}, nil
}

func (d *DeployTime) template(data any) (err error) {
	if err = d.Schema.Template(data); err != nil {
		return err
	}

	if err = d.Name.Template(data); err != nil {
		return err
	}

	if err = d.Branch.Template(data); err != nil {
		return err
	}

	if err = d.Sha.Template(data); err != nil {
		return err
	}

	if err = d.Origin.Template(data); err != nil {
		return err
	}

	if err = d.Role.Template(data); err != nil {
		return err
	}

	if err = d.Policy.Template(data); err != nil {
		return err
	}

	if err = d.Resources.Template(data); err != nil {
		return err
	}

	if err = d.Bus.Template(data); err != nil {
		return err
	}

	return nil
}
